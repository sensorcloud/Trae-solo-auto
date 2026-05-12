package mesh

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"k8s.io/klog/v2"
)

type ServiceMeshType string

const (
	ServiceMeshLinkerd ServiceMeshType = "linkerd"
	ServiceMeshIstio   ServiceMeshType = "istio"
)

type ServiceMeshConfig struct {
	Type        ServiceMeshType
	Enabled     bool
	CertPath    string
	MTLSEnabled bool
}

type Certs struct {
	CA *x509.CertPool
}

type ServiceMesh struct {
	config *ServiceMeshConfig
	certs  *Certs
}

func NewServiceMesh(config *ServiceMeshConfig) (*ServiceMesh, error) {
	if config == nil {
		config = &ServiceMeshConfig{
			Type:        ServiceMeshLinkerd,
			Enabled:     true,
			CertPath:    "/var/run/linkerd_io-identity-endpoint.crt",
			MTLSEnabled: true,
		}
	}

	sm := &ServiceMesh{
		config: config,
	}

	if err := sm.loadCerts(); err != nil {
		return nil, err
	}

	return sm, nil
}

func (sm *ServiceMesh) loadCerts() error {
	if sm.config.CertPath == "" {
		sm.config.CertPath = "/var/run/linkerd_io-identity-endpoint.crt"
	}

	certFile := sm.config.CertPath

	certBytes, err := os.ReadFile(certFile)
	if err != nil {
		klog.Warningf("Certificate file not found at %s: %v", certFile, err)
		return nil
	}

	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(certBytes); !ok {
		return fmt.Errorf("failed to append certificates to pool")
	}

	sm.certs = &Certs{
		CA: certPool,
	}

	klog.Info("Service mesh certificates loaded successfully")
	return nil
}

func (sm *ServiceMesh) GetMTLSConfig() *tls.Config {
	if !sm.config.MTLSEnabled || sm.certs == nil {
		return &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		ClientCAs:          sm.certs.CA,
		RootCAs:            sm.certs.CA,
		ClientAuth:         tls.RequireAndVerifyClientCert,
		GetCertificate:     sm.getServerCertificate,
		GetClientCertificate: sm.getClientCertificate,
	}
}

func (sm *ServiceMesh) getServerCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	if sm.certs == nil {
		return nil, fmt.Errorf("no server certificate configured")
	}
	return nil, nil
}

func (sm *ServiceMesh) getClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	if sm.certs == nil {
		return nil, fmt.Errorf("no client certificate configured")
	}
	return nil, nil
}

func (sm *ServiceMesh) InjectHeaders(_ interface{}, headers map[string]string) error {
	for key, value := range headers {
		klog.V(4).Infof("Injecting header %s=%s", key, value)
	}
	return nil
}

func (sm *ServiceMesh) GetConfig() *ServiceMeshConfig {
	return sm.config
}

func (sm *ServiceMesh) UpdateConfig(config *ServiceMeshConfig) {
	sm.config = config
}
