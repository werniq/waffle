package proxy

import (
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"waffle/internal/certificate"
	"waffle/internal/redirect"
)

var (
	ciphers = []uint16{
		// TLS 1.3
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,

		// ECDSA is about 3 times faster than RSA on the server side.
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,

		// RSA is slower on the server side but still widely used.
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,

		// Added so all ciphers are available

		tls.TLS_RSA_WITH_RC4_128_SHA,
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
		tls.TLS_FALLBACK_SCSV,
	}

	http11    = "http/1.1"
	http2     = "h2"
	http3     = "h3"
	alpnProto = "acme-tls/1"
)

type Server struct {
	addr                string
	certificateProvider certificate.Provider
	redirectHandler     *redirect.Handler
}

func NewServer(
	addr string,
	certificateProvider certificate.Provider,
	redirectHandler *redirect.Handler,
) *Server {
	return &Server{
		addr:                addr,
		certificateProvider: certificateProvider,
		redirectHandler:     redirectHandler,
	}
}

func (s *Server) Start() error {
	caCertPool, err := s.certificateProvider.GetCACertificatesPool()
	if err != nil {
		return fmt.Errorf("get ca certificates pool using certificate provider: %w", err)
	}

	serverCertificate, err := s.certificateProvider.GetTLSCertificate()
	if err != nil {
		return fmt.Errorf("get tls certificate using certificate provider: %w", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		MaxVersion:   tls.VersionTLS13,
		CipherSuites: ciphers,
		NextProtos:   []string{http3, http2, http11, alpnProto},
		Certificates: []tls.Certificate{*serverCertificate},
		ClientAuth:   tls.VerifyClientCertIfGiven,
		Rand:         rand.Reader,
		RootCAs:      caCertPool,
		ClientCAs:    caCertPool,
	}

	tcpListener, err := tls.Listen("tcp", s.addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("tls tcp listener listen: %w", err)
	}

	server := &http.Server{
		Handler:           s.redirectHandler,
		Addr:              fmt.Sprintf("%s%s", "localhost", s.addr),
		ReadHeaderTimeout: 120 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadTimeout:       120 * time.Second,
		TLSConfig:         tlsConfig,
		MaxHeaderBytes:    1048576,
		ErrorLog:          log.New(os.Stderr, "proxy server: ", 0),
	}

	if err := server.Serve(tcpListener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server serve: %w", err)
	}

	return nil
}
