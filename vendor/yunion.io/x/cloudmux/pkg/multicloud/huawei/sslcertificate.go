package huawei

import (
	"bytes"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SSSLCertificate struct {
	multicloud.SVirtualResourceBase
	HuaweiTags
	client *SHuaweiClient

	Id                 string // 证书ID
	Name               string // 证书名称
	Domain             string // 证书绑定的域名
	Sans               string // 证书的SAN（Subject Alternative Name）扩展属性，表示证书关联的其他域名、IP地址等
	SignatureAlgorithm string // 证书签名算法
	DeploySupport      bool   // 是否支持部署
	Type               string // 证书类型 取值如下: DV_SSL_CERT、DV_SSL_CERT_BASIC、EV_SSL_CERT、EV_SSL_CERT_PRO、OV_SSL_CERT、OV_SSL_CERT_PRO
	Brand              string // 证书品牌 取值如下:GLOBALSIGN、SYMANTEC、GEOTRUST、CFCA
	//ExpireTime          time.Time // 证书过期时间
	ExpireTime          string // 证书过期时间
	DomainType          string // 域名类型。取值如下: SINGLE_DOMAIN:单域名 WILDCARD:通配符 MULTI_DOMAIN:多域名
	ValidityPeriod      int    // 证书有效期，单位为月
	Status              string // 证书状态。取值如下: PAID:证书已支付;待申请证书 ISSUED:证书已签发 CHECKING:证书申请审核中 CANCELCHECKING:取消证书申请审核中 UNPASSED:证书申请未通过 EXPIRED:证书已过期 REVOKING:证书吊销申请审核中 CANCLEREVOKING:证书取消吊销申请审核中 REVOKED:证书已吊销 UPLOAD:证书托管中 SUPPLEMENTCHECKING:多域名证书新增附加域名审核中 CANCELSUPPLEMENTING:取消新增附加域名审核中
	DomainCount         int    // 证书绑定的域名数量
	WildcardCount       int    // 证书绑定的通配符域名数量
	Description         string // 证书描述
	EnterpriseProjectId string // 企业项目ID 默认为“0”

	// certificate details
	detailsInitd bool
	Certificate  string `json:"certificate"` // 证书内容
	PrivateKey   string `json:"private_key"` // 证书私钥
}

func (s *SSSLCertificate) GetSans() string {
	return s.Sans
}

func (s *SSSLCertificate) GetStartDate() time.Time {
	return s.GetEndDate().AddDate(0, -s.ValidityPeriod, 0)
}

func (s *SSSLCertificate) GetProvince() string {
	return ""
}

func (s *SSSLCertificate) GetCommon() string {
	return s.Domain
}

func (s *SSSLCertificate) GetCountry() string {
	return ""
}

func (s *SSSLCertificate) GetIssuer() string {
	return s.Brand
}

func (s *SSSLCertificate) GetExpired() bool {
	return time.Now().After(s.GetEndDate())
}

func (s *SSSLCertificate) GetEndDate() time.Time {
	t, _ := time.Parse("2006-01-02 15:04:05", s.ExpireTime)
	return t
}

func (s *SSSLCertificate) GetFingerprint() string {
	var buf bytes.Buffer
	s.GetDetails()
	certBlock, _ := pem.Decode([]byte(s.Certificate))
	if certBlock == nil {
		return ""
	}
	cert, _ := x509.ParseCertificate(certBlock.Bytes)
	fingerprint := sha1.Sum(cert.Raw)
	for _, f := range fingerprint {
		fmt.Fprintf(&buf, "%02X", f)
	}
	return buf.String()
}

func (s *SSSLCertificate) GetCity() string {
	return ""
}

func (s *SSSLCertificate) GetOrgName() string {
	return ""
}

func (s *SSSLCertificate) GetId() string {
	return s.Id
}

func (s *SSSLCertificate) GetName() string {
	return s.Name
}

func (s *SSSLCertificate) GetGlobalId() string {
	return s.Id
}

func (s *SSSLCertificate) GetStatus() string {
	if s.GetExpired() {
		return "expired"
	} else {
		return "normal"
	}
}

func (s *SSSLCertificate) GetIsUpload() bool {
	if s.Status == "UPLOAD" {
		return true
	}
	return false
}

func (s *SSSLCertificate) GetCert() string {
	s.GetDetails()
	return s.Certificate
}

func (s *SSSLCertificate) GetKey() string {
	s.GetDetails()
	return s.PrivateKey
}

func (s *SSSLCertificate) GetDetails() (*SSSLCertificate, error) {
	if !s.detailsInitd {
		cert, err := s.client.GetISSLCertificate(s.GetId())
		if err != nil {
			return nil, err
		}
		s.detailsInitd = true
		_cert, ok := cert.(*SSSLCertificate)
		if !ok {
			return nil, errors.Wrapf(err, "cert.(*SSSLCertificate)")
		}
		s.Certificate = _cert.Certificate
		s.PrivateKey = _cert.PrivateKey
	}
	return s, nil
}

func (r *SHuaweiClient) GetSSLCertificates(size, offset int) ([]SSSLCertificate, int, error) {
	if size < 1 || size > 50 {
		size = 50
	}
	if offset < 0 {
		offset = 0
	}

	params := url.Values{
		"limit":    []string{fmt.Sprintf("%d", size)},
		"offset":   []string{fmt.Sprintf("%d", offset)},
		"sort_key": []string{"certExpiredTime"},
		"sort_dir": []string{"DESC"},
	}
	resp, err := r.sslcertList(HUAWEI_CERT_DEFAULT_REGION, "scm/certificates", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "CertificateList")
	}

	ret := make([]SSSLCertificate, 0)
	err = resp.Unmarshal(&ret, "certificates")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}

	totalCount, _ := resp.Int("total_count")
	return ret, int(totalCount), nil
}

func (r *SHuaweiClient) GetSSLCertificate(certId string) (*SSSLCertificate, error) {
	resp, err := r.sslcertExport(HUAWEI_CERT_DEFAULT_REGION, "scm/certificates", certId)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeCertificateDetail")
	}

	cert := &SSSLCertificate{}
	err = resp.Unmarshal(cert)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	cert.client = r
	return cert, nil
}
