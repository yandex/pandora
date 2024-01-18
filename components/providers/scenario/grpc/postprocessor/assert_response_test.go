package postprocessor

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

type testProto struct {
	proto.Message
	str string
}

func (t testProto) String() string {
	return t.str
}

func TestAssertResponse_Process(t *testing.T) {
	type fields struct {
		Payload    []string
		StatusCode int
	}
	type args struct {
		msg  proto.Message
		code int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Valid Response",
			fields: fields{
				Payload:    []string{"Hello, World!"},
				StatusCode: http.StatusOK,
			},
			args: args{
				msg:  testProto{str: "Hello, World!"},
				code: 200,
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid Payload",
			fields: fields{
				Payload:    []string{"Invalid Text"},
				StatusCode: http.StatusOK,
			},
			args: args{
				msg:  testProto{str: "Hello, World!"},
				code: 200,
			},
			wantErr: assert.Error,
		},
		{
			name: "Empty Payload",
			fields: fields{
				Payload:    []string{"Hello, World!"},
				StatusCode: http.StatusOK,
			},
			args: args{
				msg:  nil,
				code: 200,
			},
			wantErr: assert.Error,
		},
		{
			name: "Empty Payload",
			fields: fields{
				Payload:    []string{},
				StatusCode: http.StatusOK,
			},
			args: args{
				msg:  nil,
				code: 200,
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid StatusCode",
			fields: fields{
				Payload:    []string{"Hello, World!"},
				StatusCode: http.StatusOK,
			},
			args: args{
				msg:  testProto{str: "Hello, World!"},
				code: 404,
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AssertResponse{
				Payload:    tt.fields.Payload,
				StatusCode: tt.fields.StatusCode,
			}
			process, err := a.Process(tt.args.msg, tt.args.code)
			assert.Nil(t, process)
			tt.wantErr(t, err, fmt.Sprintf("Process(%v, %v)", tt.args.msg, tt.args.code))
		})
	}
}
