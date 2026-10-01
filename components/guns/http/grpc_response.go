package phttp

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/yandex/pandora/components/grpcstatus"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/lib/answlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

type grpcResponse struct {
	code       codes.Code
	status     string
	messageRaw string
	detailsBin string
	details    *statuspb.Status
	source     string
}

func readGRPCResponse(res *http.Response, bodyBytes int64) grpcResponse {
	metadata := res.Trailer
	source := "trailers"
	if bodyBytes == 0 && res.ContentLength == 0 &&
		len(res.Trailer) == 0 && endedAtResponseHeaders(res.Body) {
		// A trailers-only response carries the final status in its sole HEADERS block.
		metadata = res.Header
		source = "trailers-only"
	}
	statuses := metadata.Values("Grpc-Status")
	detailValues := metadata.Values("Grpc-Status-Details-Bin")
	result := grpcResponse{
		code:       codes.Unknown,
		status:     strings.Join(statuses, ","),
		messageRaw: metadata.Get("Grpc-Message"),
		detailsBin: strings.Join(detailValues, ","),
		source:     source,
	}
	grpcContentType := isGRPCContentType(res.Header.Get("Content-Type"))
	// Match the bundled grpc-go v1.62 client: non-200 HTTP is a transport failure even with grpc-status: 0.
	if !grpcContentType || res.StatusCode != http.StatusOK {
		result.code = grpcCodeFromHTTP(res.StatusCode)
		result.source = "non-grpc-http-fallback"
		if grpcContentType {
			result.source = "non-200-http-fallback"
		}
		return result
	}
	var singleValue string
	var singleData []byte
	valueCount := 0
	for _, field := range detailValues {
		for {
			value, rest, hasMore := strings.Cut(field, ",")
			data, decodeErr := decodeGRPCBinaryValue(value)
			if decodeErr != nil {
				result.code = codes.Internal
				result.source = "invalid-status-details"
				return result
			}
			valueCount++
			if valueCount == 1 {
				singleValue, singleData = value, data
			}
			if !hasMore {
				break
			}
			field = rest
		}
	}
	if len(statuses) == 0 {
		result.code = grpcCodeFromHTTP(res.StatusCode)
		result.source = "http-fallback"
		return result
	}
	if len(statuses) != 1 || statuses[0] == "" {
		result.source = "invalid-grpc-status"
		return result
	}
	for _, digit := range statuses[0] {
		if digit < '0' || digit > '9' {
			result.source = "invalid-grpc-status"
			return result
		}
	}
	code, err := strconv.ParseInt(statuses[0], 10, 32)
	if err != nil {
		result.source = "invalid-grpc-status"
		return result
	}
	result.code = codes.Code(code)
	if valueCount == 1 && singleValue != "" {
		var details statuspb.Status
		if proto.Unmarshal(singleData, &details) == nil {
			result.details = &details
			if statusDetailsHasCode(singleData) && details.Code != int32(result.code) {
				result.code = codes.Internal
				result.source = "conflicting-status-details"
			}
		}
	}
	return result
}

func decodeGRPCBinaryValue(value string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return base64.RawStdEncoding.DecodeString(value)
	}
	return data, nil
}

func (r grpcResponse) detailsString() string {
	if r.details != nil {
		return r.details.String()
	}
	if r.detailsBin == "" || strings.Contains(r.detailsBin, ",") {
		return ""
	}
	data, err := decodeGRPCBinaryValue(r.detailsBin)
	if err != nil {
		return ""
	}
	var details statuspb.Status
	if proto.Unmarshal(data, &details) != nil {
		return ""
	}
	return details.String()
}

func statusDetailsHasCode(data []byte) bool {
	for len(data) != 0 {
		number, wireType, tagLen := protowire.ConsumeTag(data)
		if tagLen < 0 {
			return false
		}
		data = data[tagLen:]
		fieldLen := protowire.ConsumeFieldValue(number, wireType, data)
		if fieldLen < 0 {
			return false
		}
		if number == 1 && wireType == protowire.VarintType {
			return true
		}
		data = data[fieldLen:]
	}
	return false
}

func isGRPCContentType(value string) bool {
	base := strings.ToLower(strings.TrimSpace(strings.SplitN(value, ";", 2)[0]))
	return base == "application/grpc" || strings.HasPrefix(base, "application/grpc+") && len(base) > len("application/grpc+")
}

func decodeGRPCMessage(encoded string) string {
	var safe strings.Builder
	safe.Grow(len(encoded))
	for i := 0; i < len(encoded); i++ {
		if encoded[i] == '%' && (i+2 >= len(encoded) || !isHex(encoded[i+1]) || !isHex(encoded[i+2])) {
			safe.WriteString("%25")
		} else {
			safe.WriteByte(encoded[i])
		}
	}
	decoded, err := url.PathUnescape(safe.String())
	if err != nil {
		return encoded
	}
	return decoded
}

func isHex(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F'
}

func grpcCodeFromHTTP(httpCode int) codes.Code {
	switch httpCode {
	case http.StatusBadRequest:
		return codes.Internal
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.Unimplemented
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return codes.Unavailable
	default:
		return codes.Unknown
	}
}

func (b *BaseGun) processGRPCResponse(req *http.Request, res *http.Response, sample *netsample.Sample) error {
	defer res.Body.Close()
	bodyBytes, readErr := io.Copy(io.Discard, res.Body)
	grpcRes := grpcResponse{code: codes.Unknown, source: "body-read-error"}
	if readErr == nil {
		grpcRes = readGRPCResponse(res, bodyBytes)
	}
	b.setGRPCResponseBytes(sample, res, bodyBytes)
	sample.SetUserProto(grpcstatus.ToHTTPCode(grpcRes.code))
	b.grpcAnswLogging(req, res, grpcRes, readErr)
	if readErr != nil {
		b.Log.Warn("Body read fail", zap.Error(readErr))
	}
	return readErr
}

func (b *BaseGun) setGRPCResponseBytes(sample *netsample.Sample, res *http.Response, bodyBytes int64) {
	if !b.Config.HTTPTrace.DumpEnabled {
		return
	}
	responseDump, err := httputil.DumpResponse(res, false)
	if err != nil {
		b.Log.Error("DumpResponse error", zap.Error(err))
		sample.SetResponseBytes(int(bodyBytes))
		return
	}
	sample.SetResponseBytes(len(responseDump) + int(bodyBytes) + trailerFieldsLen(res.Trailer))
}

func (b *BaseGun) grpcAnswLogging(req *http.Request, res *http.Response, grpcRes grpcResponse, readErr error) {
	b.AnswLog.Report("REQUEST/RESPONSE", []zapcore.Field{zap.Int(answlog.FilterAndSampleGroup, int(grpcRes.code))}, func() []zapcore.Field {
		request := req
		includeRequestBody := false
		if req.GetBody != nil {
			if body, getBodyErr := req.GetBody(); getBodyErr == nil {
				defer body.Close()
				request = req.Clone(req.Context())
				request.Body = body
				includeRequestBody = true
			}
		}
		reqDump, dumpErr := httputil.DumpRequestOut(request, includeRequestBody)
		if dumpErr != nil {
			reqDump = fmt.Appendf(nil, "Error dumping request: %s", dumpErr)
		}
		respDump, dumpErr := httputil.DumpResponse(res, false)
		if dumpErr != nil {
			respDump = fmt.Appendf(nil, "Error dumping response: %s", dumpErr)
		}
		trailerDump := dumpTrailerFields(res.Trailer)
		if len(trailerDump) != 0 {
			respDump = append(respDump, []byte("Trailers:\r\n")...)
			respDump = append(respDump, trailerDump...)
		}
		fields := []zapcore.Field{
			zap.String("req", string(reqDump)),
			zap.String("resp", string(respDump)),
			zap.Int("http_status", res.StatusCode),
			zap.String("grpc_status_source", grpcRes.source),
			zap.String("grpc_status_raw", grpcRes.status),
			zap.String("grpc_message", decodeGRPCMessage(grpcRes.messageRaw)),
			zap.String("grpc_status_details_bin", grpcRes.detailsBin),
			zap.String("grpc_status_details", grpcRes.detailsString()),
		}
		if readErr != nil {
			fields = append(fields, zap.String("read_error", readErr.Error()))
		}
		return fields
	})
}

func trailerFieldsLen(fields http.Header) int {
	var length int
	for name, values := range fields {
		for _, value := range values {
			length += len(name) + len(value) + len(": \r\n")
		}
	}
	return length
}

func dumpTrailerFields(fields http.Header) []byte {
	keys := make([]string, 0, len(fields))
	for name := range fields {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	var dump []byte
	for _, name := range keys {
		for _, value := range fields[name] {
			dump = fmt.Appendf(dump, "%s: %s\r\n", name, value)
		}
	}
	return dump
}
