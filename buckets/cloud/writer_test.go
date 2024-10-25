package cloud

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	cpi "github.com/hsuanshao/go-tools/buckets/cloud/provider/interface/mocks"
	e "github.com/hsuanshao/go-tools/buckets/entity"
	ifc "github.com/hsuanshao/go-tools/buckets/interface"
	"github.com/hsuanshao/go-tools/ctx"
)

var (
	mockCTX      = ctx.Background()
	mockObjSPSrv = new(cpi.ObjectServiceProvider)
)

type wtestSuite struct {
	suite.Suite
	CloudWriter ifc.BucketWriter
}

func TestNewWriter(t *testing.T) {
	mockConfig := e.Config{
		CSP:    e.AWS,
		Bucket: "btq.ag",
		Region: "ap-northeast-1",
	}

	_, err := NewWriter(mockCTX, &mockConfig)
	if err != nil {
		t.Error(err)
	}

	mockConf2 := e.Config{
		CSP:    e.AWS,
		Bucket: "btq.ag",
		Region: "",
	}
	_, err = NewWriter(mockCTX, &mockConf2)
	if err != e.ErrS3RegionisEmptyStr {
		t.Error(err)
	}

	mockConfig3 := e.Config{
		CSP:    e.AWS,
		Bucket: "",
		Region: "ap-northeast-1",
	}

	_, err = NewWriter(mockCTX, &mockConfig3)
	if err != e.ErrS3BucketisEmptyStr {
		t.Error(err)
	}
}

func TestWriterSuite(t *testing.T) {
	ctx.SetDebugLevel()
	suite.Run(t, new(wtestSuite))
}

func (ws *wtestSuite) SetupTest() {
	ws.CloudWriter = &writeImpl{
		cloud: mockObjSPSrv,
	}
}

func (ws *wtestSuite) TearDownSuite() {
	ws.CloudWriter = &writeImpl{
		cloud: mockObjSPSrv,
	}
	mockCTX = ctx.Background()
}

func (ws *wtestSuite) TestUpload() {
	testcases := []struct {
		Case            string
		MockFunc        func()
		Mime            e.ContentType
		ObjPath         string
		ObjContent      []byte
		ObjMetadata     map[string]string
		ExpObjURL       string
		ExpPresignedURL string
		ExpErr          error
	}{
		{
			Case: "Normal case",
			MockFunc: func() {
				// mockObjSPSrv.On("GetObjectAttributes", &awsS3.GetObjectAttributesInput{Bucket: aws.String("btq-bucket"), Key: aws.String("i18n/ja-JP.json")}).Return(&awsS3.GetObjectAttributesOutput{LastModified: nil}, nil).Once()
				mockObjSPSrv.On("Upload", mockCTX, e.JSON, "i18n/ja-JP.json", []byte(`{"menu":"機能メニュー","prdouct":"プロダクツ"}`), map[string]string{"btq-product": "official-website", "version": "v1.1", "environment": "production", "language": "ja-JP"}).Return("https://btq-bucket.s3.amazonaws.com/i18n/ja-JP.json", "https://btq-bucket.s3.amazonaws.com/i18n/ja-JP.json?X-Amz-Algorithm=AW54-HMAC-SHA256&X-Amz-Date=20221130T150956&X-Amz-SignedHeaders=host&X-Amz-Expires=300&X-Amz-Credential=AKIAYIDZSD2%202211%2Fap-northeast-1%2Fs3%2Faws4_request&aws4_request&X-amz-Signature=062ddg453248sdf3298df98u3984dsasd34sd", nil).Once()
			},
			Mime:            e.JSON,
			ObjPath:         "i18n/ja-JP.json",
			ObjContent:      []byte(`{"menu":"機能メニュー","prdouct":"プロダクツ"}`),
			ObjMetadata:     map[string]string{"btq-product": "official-website", "version": "v1.1", "environment": "production", "language": "ja-JP"},
			ExpObjURL:       "https://btq-bucket.s3.amazonaws.com/i18n/ja-JP.json",
			ExpPresignedURL: "https://btq-bucket.s3.amazonaws.com/i18n/ja-JP.json?X-Amz-Algorithm=AW54-HMAC-SHA256&X-Amz-Date=20221130T150956&X-Amz-SignedHeaders=host&X-Amz-Expires=300&X-Amz-Credential=AKIAYIDZSD2%202211%2Fap-northeast-1%2Fs3%2Faws4_request&aws4_request&X-amz-Signature=062ddg453248sdf3298df98u3984dsasd34sd",
			ExpErr:          nil,
		},
		{
			Case: "upload a file, but the path already have an object",
			MockFunc: func() {
				mockObjSPSrv.On("Upload", mockCTX, e.JSON, "i18n/zh-TW.json", []byte(`{"menu":"選單","prdouct":"產品"}`), map[string]string{"btq-product": "official-website", "version": "v1.1", "environment": "production", "language": "zh-TW"}).Return("", "", e.ErrObjectPathHasItem).Once()
			},
			Mime:            e.JSON,
			ObjPath:         "i18n/zh-TW.json",
			ObjContent:      []byte(`{"menu":"選單","prdouct":"產品"}`),
			ObjMetadata:     map[string]string{"btq-product": "official-website", "version": "v1.1", "environment": "production", "language": "zh-TW"},
			ExpObjURL:       "",
			ExpPresignedURL: "",
			ExpErr:          e.ErrUploadObj,
		},
	}

	for idx, c := range testcases {
		mockCTX = ctx.WithValue(mockCTX, "case no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		objURL, presignedURL, err := ws.CloudWriter.Upload(mockCTX, c.Mime, c.ObjPath, c.ObjContent, c.ObjMetadata)

		if !ws.Equal(c.ExpObjURL, objURL) {
			mockCTX.Error("expected upload object return url not match")
		}

		if !ws.Equal(c.ExpPresignedURL, presignedURL) {
			mockCTX.Error("expected presigned url not match")
		}

		ws.Equal(c.ExpErr, err, fmt.Sprintf("case[%v]%v:expected error output not match", idx, c.Case))
	}
}

func (ws *wtestSuite) TestUpdate() {
	testcases := []struct {
		Case        string
		MockFunc    func()
		Mime        e.ContentType
		ObjPath     string
		ObjContent  []byte
		ObjMetadata map[string]string
		ExpObjURL   string
		ExpErr      error
	}{
		{
			Case: "normal case",
			MockFunc: func() {
				mockObjSPSrv.On("Override", mockCTX, e.JSON, "i18n/zh-TW.json", []byte(`{"menu":"選單","prdouct":"產品","Company":關於BTQ}`), map[string]string{"btq-product": "official-website", "version": "v1.2", "environment": "production", "language": "zh-TW"}).Return("https://btq-bucket.s3.amazonaws.com/i18n/zh-TW.json", nil).Once()
			},
			Mime:        e.JSON,
			ObjPath:     "i18n/zh-TW.json",
			ObjContent:  []byte(`{"menu":"選單","prdouct":"產品","Company":關於BTQ}`),
			ObjMetadata: map[string]string{"btq-product": "official-website", "version": "v1.2", "environment": "production", "language": "zh-TW"},
			ExpObjURL:   "https://btq-bucket.s3.amazonaws.com/i18n/zh-TW.json",
			ExpErr:      nil,
		},
		{
			Case: "override object not exists case",
			MockFunc: func() {
				mockObjSPSrv.On("Override", mockCTX, e.JSON, "i18n/zhtw.json", []byte(`{"menu":"選單","prdouct":"產品","Company":關於BTQ}`), map[string]string{"btq-product": "official-website", "version": "v1.2", "environment": "production", "language": "zh-TW"}).Return("", e.ErrFetchOriginObjFromS3ByGivenObjPath).Once()
			},
			Mime:        e.JSON,
			ObjPath:     "i18n/zhtw.json",
			ObjContent:  []byte(`{"menu":"選單","prdouct":"產品","Company":關於BTQ}`),
			ObjMetadata: map[string]string{"btq-product": "official-website", "version": "v1.2", "environment": "production", "language": "zh-TW"},
			ExpObjURL:   "",
			ExpErr:      e.ErrOverrideObj,
		},
	}

	for idx, c := range testcases {
		mockCTX = ctx.WithValue(mockCTX, "case no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		objURL, err := ws.CloudWriter.Update(mockCTX, c.Mime, c.ObjPath, c.ObjContent, c.ObjMetadata)

		if !ws.Equal(c.ExpObjURL, objURL) {
			mockCTX.Error("expected upload object return url not match")
		}

		ws.Equal(c.ExpErr, err, fmt.Sprintf("case[%v]%v:expected error output not match", idx, c.Case))
	}
}

func (ws *wtestSuite) TestGeneratePutPresignedURL() {
	testcases := []struct {
		Case               string
		MockFunc           func()
		Mime               e.ContentType
		ObjPath            string
		LimitDuration      time.Duration
		ObjMetadata        map[string]string
		ExpPutPresignedURL string
		ExpErr             error
	}{
		{
			Case: "success cases",
			MockFunc: func() {
				mockObjSPSrv.On("PutPresignedURL", mockCTX, "client_upload/massa/2022/11/30/fgds2435342.toml", e.TOML, 10*time.Minute, map[string]string{}).Return("https://btq-bucket.s3.amazonaws.com/client_upload/massa/2022/11/30/fgds2435342.toml?X-Amz-Algorithm=AW54-HMAC-SHA256&X-Amz-Date=20221130T150956&X-Amz-SignedHeaders=host&X-Amz-Expires=300&X-Amz-Credential=AKIAYIDZSD2%202211%2Fap-northeast-1%2Fs3%2Faws4_request&aws4_request&X-amz-Signature=062ddg453248sdf3298df98u3984dsasd34sd", nil).Once()
			},
			ObjPath:            "client_upload/massa/2022/11/30/fgds2435342.toml",
			Mime:               e.TOML,
			LimitDuration:      10 * time.Minute,
			ObjMetadata:        map[string]string{},
			ExpPutPresignedURL: "https://btq-bucket.s3.amazonaws.com/client_upload/massa/2022/11/30/fgds2435342.toml?X-Amz-Algorithm=AW54-HMAC-SHA256&X-Amz-Date=20221130T150956&X-Amz-SignedHeaders=host&X-Amz-Expires=300&X-Amz-Credential=AKIAYIDZSD2%202211%2Fap-northeast-1%2Fs3%2Faws4_request&aws4_request&X-amz-Signature=062ddg453248sdf3298df98u3984dsasd34sd",
			ExpErr:             nil,
		},
		{
			Case: "failed cases",
			MockFunc: func() {
				mockObjSPSrv.On("PutPresignedURL", mockCTX, "https://s3-ap-east-1.amazonaws.com/btq-bucket/client_upload/massa/2022/11/30/fgds2435342.toml", e.TOML, 10*time.Minute, map[string]string{}).Return("", e.ErrWithoutPermissionToAccess).Once()
			},
			ObjPath:            "https://s3-ap-east-1.amazonaws.com/btq-bucket/client_upload/massa/2022/11/30/fgds2435342.toml",
			Mime:               e.TOML,
			LimitDuration:      10 * time.Minute,
			ObjMetadata:        map[string]string{},
			ExpPutPresignedURL: "",
			ExpErr:             e.ErrGenPutPresignedURL,
		},
	}

	for idx, c := range testcases {
		mockCTX = ctx.WithValue(mockCTX, "case no", idx)
		mockCTX = ctx.WithValue(mockCTX, "case name", c.Case)
		c.MockFunc()

		objURL, err := ws.CloudWriter.GeneratePutPresignedURL(mockCTX, c.LimitDuration, c.Mime, c.ObjPath, c.ObjMetadata)

		if !ws.Equal(c.ExpPutPresignedURL, objURL) {
			mockCTX.Error("expected upload object return url not match")
		}

		ws.Equal(c.ExpErr, err, fmt.Sprintf("case[%v]%v:expected error output not match", idx, c.Case))
	}
}
