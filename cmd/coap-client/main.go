package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"os"
	"time"

	piondtls "github.com/pion/dtls/v2"
	"github.com/pion/dtls/v2/examples/util"
	"github.com/plgd-dev/go-coap/v2/dtls"
	"github.com/plgd-dev/go-coap/v2/message"
)

func main() {
	certificate, err := util.LoadKeyAndCertificate("certs/client-key.pem",
		"certs/client.pem")
	util.Check(err)

	rootCertificate, err := util.LoadCertificate("certs/server.pem")
	util.Check(err)
	certPool := x509.NewCertPool()
	cert, err := x509.ParseCertificate(rootCertificate.Certificate[0])
	util.Check(err)
	certPool.AddCert(cert)

	co, err := dtls.Dial("127.0.0.1:5689", &piondtls.Config{
		Certificates:         []tls.Certificate{*certificate},
		ExtendedMasterSecret: piondtls.RequireExtendedMasterSecret,
		RootCAs:              certPool,
	})
	if err != nil {
		log.Fatalf("Error dialing: %v", err)
	}
	path := "d/124/s/temp"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := co.Post(ctx, path, message.TextPlain, bytes.NewReader([]byte("32.5")))
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	log.Printf("Response payload: %+v", resp)
}
