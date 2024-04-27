package phttp

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	ammomock "github.com/yandex/pandora/components/guns/http/mocks"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/coretest"
	"github.com/yandex/pandora/core/engine"
	"github.com/yandex/pandora/lib/monitoring"
	"github.com/yandex/pandora/lib/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger() *zap.Logger {
	zapConf := zap.NewDevelopmentConfig()
	zapConf.Level.SetLevel(zapcore.DebugLevel)
	log, err := zapConf.Build(zap.AddCaller())
	if err != nil {
		zap.L().Fatal("Logger build failed", zap.Error(err))
	}
	return log
}

func newEngineMetrics(prefix string) engine.Metrics {
	return engine.Metrics{
		Request:        monitoring.NewCounter(prefix + "_Requests"),
		Response:       monitoring.NewCounter(prefix + "_Responses"),
		InstanceStart:  monitoring.NewCounter(prefix + "_UsersStarted"),
		InstanceFinish: monitoring.NewCounter(prefix + "_UsersFinished"),
	}
}

func TestGunSuite(t *testing.T) {
	suite.Run(t, new(BaseGunSuite))
}

type BaseGunSuite struct {
	suite.Suite
	fs      afero.Fs
	log     *zap.Logger
	metrics engine.Metrics
	base    BaseGun
	gunDeps core.GunDeps
}

func (s *BaseGunSuite) SetupSuite() {
	s.log = testutil.NewLogger()
	s.metrics = newEngineMetrics("http_suite")
}

func (s *BaseGunSuite) SetupTest() {
	s.base = BaseGun{Config: DefaultHTTPGunConfig()}
}

func (s *BaseGunSuite) Test_BindResultTo_Panics() {
	s.Run("nil panic", func() {
		s.Panics(func() {
			_ = s.base.Bind(nil, testDeps())
		})
	})
	s.Run("nil panic", func() {
		res := &netsample.TestAggregator{}
		_ = s.base.Bind(res, testDeps())
		s.Require().Equal(res, s.base.Aggregator)
		s.Panics(func() {
			_ = s.base.Bind(&netsample.TestAggregator{}, testDeps())
		})
	})
}

type ammoMock struct {
	requestCallCnt   int
	idCallCnt        int
	isInvalidCallCnt int
}

func (a *ammoMock) Request() (*http.Request, *netsample.Sample) {
	a.requestCallCnt++
	return nil, nil
}

func (a *ammoMock) ID() uint64 {
	a.idCallCnt++
	return 0
}

func (a *ammoMock) IsInvalid() bool {
	a.isInvalidCallCnt++
	return false
}

type testDecoratedClient struct {
	client    Client
	t         *testing.T
	before    func(req *http.Request)
	after     func(req *http.Request, res *http.Response, err error)
	returnRes *http.Response
	returnErr error
}

func (c *testDecoratedClient) Do(req *http.Request) (*http.Response, error) {
	if c.before != nil {
		c.before(req)
	}
	if c.client == nil {
		return c.returnRes, c.returnErr
	}
	res, err := c.client.Do(req)
	if c.after != nil {
		c.after(req, res, err)
	}
	return res, err
}

func (c *testDecoratedClient) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}

func (s *BaseGunSuite) Test_Shoot_BeforeBindPanics() {
	s.base.Client = &testDecoratedClient{
		client: s.base.Client,
		before: func(req *http.Request) { panic("should not be called\"") },
		after:  nil,
	}
	am := &ammoMock{}

	s.Panics(func() {
		s.base.Shoot(am)
	})
}

func (s *BaseGunSuite) Test_Shoot() {
	var (
		body io.ReadCloser

		am       *ammomock.Ammo
		req      *http.Request
		tag      string
		res      *http.Response
		sample   *netsample.Sample
		results  *netsample.TestAggregator
		shootErr error
	)
	beforeEach := func() {
		am = ammomock.NewAmmo(s.T())
		am.On("IsInvalid").Return(false).Maybe()
		req = httptest.NewRequest("GET", "/1/2/3/4", nil)
		tag = ""
		results = &netsample.TestAggregator{}
		_ = s.base.Bind(results, testDeps())
	}

	justBeforeEach := func() {
		sample = netsample.Acquire(tag)
		am.On("Request").Return(req, sample).Maybe()
		res = &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       ioutil.NopCloser(body),
			Request:    req,
		}
		s.base.Shoot(am)
		s.Require().Len(results.Samples, 1)
		shootErr = results.Samples[0].Err()
	}

	s.Run("Do ok", func() {
		beforeEachDoOk := func() {
			body = ioutil.NopCloser(strings.NewReader("aaaaaaa"))
			s.base.AnswLog = zap.NewNop()
			s.base.Client = &testDecoratedClient{
				before: func(doReq *http.Request) {
					s.Require().Equal(req, doReq)
				},
				returnRes: &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       ioutil.NopCloser(body),
					Request:    req,
				},
			}
		}
		s.Run("ammo sample sent to results", func() {
			s.SetupTest()
			beforeEach()
			beforeEachDoOk()
			justBeforeEach()

			s.Assert().Len(results.Samples, 1)
			s.Assert().Equal(sample, results.Samples[0])
			s.Assert().Equal("__EMPTY__", sample.Tags())
			s.Assert().Equal(res.StatusCode, sample.ProtoCode())
			_ = shootErr
		})

		s.Run("body read well", func() {
			s.SetupTest()
			beforeEach()
			beforeEachDoOk()
			justBeforeEach()

			s.Assert().NoError(shootErr)
			_, err := body.Read([]byte{0})
			s.Assert().ErrorIs(err, io.EOF, "body should be read fully")
		})

		s.Run("autotag options is set", func() {
			beforeEacAautotag := (func() { s.base.Config.AutoTag.Enabled = true })

			s.Run("autotagged", func() {
				s.SetupTest()
				beforeEach()
				beforeEachDoOk()
				beforeEacAautotag()
				justBeforeEach()

				s.Assert().Equal("/1/2", sample.Tags())
			})

			s.Run("tag is already set", func() {
				const presetTag = "TAG"
				beforeEachTagIsAlreadySet := func() { tag = presetTag }
				s.Run("no tag added", func() {
					s.SetupTest()
					beforeEach()
					beforeEachDoOk()
					beforeEacAautotag()
					beforeEachTagIsAlreadySet()
					justBeforeEach()

					s.Assert().Equal(presetTag, sample.Tags())
				})

				s.Run("no-tag-only set to false", func() {
					beforeEachNoTagOnly := func() { s.base.Config.AutoTag.NoTagOnly = false }
					s.Run("autotag added", func() {
						s.SetupTest()
						beforeEach()
						beforeEachDoOk()
						beforeEacAautotag()
						beforeEachTagIsAlreadySet()
						beforeEachNoTagOnly()
						justBeforeEach()

						s.Assert().Equal(presetTag+"|/1/2", sample.Tags())
					})
				})
			})
		})

		s.Run("Connect set", func() {
			var connectCalled, doCalled bool
			beforeEachConnectSet := func() {
				s.base.Connect = func(ctx context.Context) error {
					connectCalled = true
					return nil
				}

				s.base.Client = &testDecoratedClient{
					client: s.base.Client,
					before: func(doReq *http.Request) {
						doCalled = true
					},
				}
			}
			s.Run("Connect called", func() {
				s.SetupTest()
				beforeEach()
				beforeEachDoOk()
				beforeEachConnectSet()
				justBeforeEach()

				s.Assert().NoError(shootErr)
				s.Assert().True(connectCalled)
				s.Assert().True(doCalled)
			})
		})
		s.Run("Connect failed", func() {
			connectErr := errors.New("connect error")
			beforeEachConnectFailed := func() {
				s.base.Connect = func(ctx context.Context) error {
					// Connect should report fail in sample itself.
					s := netsample.Acquire("")
					s.SetErr(connectErr)
					results.Report(s)
					return connectErr
				}
			}
			s.Run("Shoot failed", func() {
				s.SetupTest()
				beforeEach()
				beforeEachDoOk()
				beforeEachConnectFailed()
				justBeforeEach()

				s.Assert().Error(shootErr)
				s.Assert().ErrorIs(shootErr, connectErr)
			})
		})
	})
}

func Test_Autotag(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		depth int
		tag   string
	}{
		{"empty", "", 2, ""},
		{"root", "/", 2, "/"},
		{"exact depth", "/1/2", 2, "/1/2"},
		{"more depth", "/1/2", 3, "/1/2"},
		{"less depth", "/1/2", 1, "/1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			URL := &url.URL{Path: tt.path}
			got := autotag(tt.depth, URL)
			assert.Equal(t, got, tt.tag)
		})
	}
}

func Test_ConfigDecode(t *testing.T) {
	var conf GunConfig
	coretest.DecodeAndValidateT(t, `
target: localhost:80
auto-tag:
  enabled: true
  uri-elements: 3
  no-tag-only: false
`, &conf)
}

func testDeps() core.GunDeps {
	return core.GunDeps{Log: testutil.NewLogger(), Ctx: context.Background()}
}
