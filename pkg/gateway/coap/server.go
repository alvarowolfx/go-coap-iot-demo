package coap

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"com.aviebrantz.coap-demo/pkg/config"
	"com.aviebrantz.coap-demo/pkg/util"
	"github.com/jeremywohl/flatten"
	"github.com/nqd/flat"
	"github.com/pion/dtls/v2"
	coap "github.com/plgd-dev/go-coap/v2"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"

	"github.com/fxamacker/cbor/v2"
	"gocloud.dev/pubsub"

	"github.com/apex/log"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

type CoAPGateway struct {
	devices   sync.Map
	router    *mux.Router
	dataTopic *pubsub.Topic
	logger    *log.Entry
	port      int
	tlsPort   int
}

func NewGateway(dataTopic *pubsub.Topic, config *config.GatewayConfig) *CoAPGateway {
	router := mux.NewRouter()
	logger := log.WithField("module", "coap-gateway")
	return &CoAPGateway{
		logger:    logger,
		port:      config.Port,
		tlsPort:   config.SslPort,
		router:    router,
		dataTopic: dataTopic,
	}
}

// Middleware function, which will be called for each request.
func (cg *CoAPGateway) routerMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		startTime := time.Now()
		ctx, err := tag.New(context.Background(), tag.Insert(KeyMethod, r.Code.String()))
		if err != nil {
			cg.logger.Errorf("err creating metric for request %v", err)
		}
		defer func() {
			stats.Record(ctx, MLatencyMs.M(sinceInMilliseconds(startTime)))
			stats.Record(ctx, MRequests.M(1))
		}()

		path, err := r.Options.Path()
		if err == nil && strings.HasPrefix(path, "d/") {
			if r.Code == codes.POST {
				cg.handlePostState(ctx, w, r)
			}
		}
		next.ServeCOAP(w, r)
	})
}

func (cg *CoAPGateway) registerClient(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		cg.logger.Infof("Registering client %v \n", w.Client().RemoteAddr())
		path, err := r.Options.Path()
		if err != nil {
			next.ServeCOAP(w, r)
			return
		}
		deviceID, err := getDeviceIDFromPath(path)
		if err != nil {
			next.ServeCOAP(w, r)
			return
		}

		cg.devices.Store(deviceID, w.Client().RemoteAddr())

		next.ServeCOAP(w, r)
	})
}

func getDeviceIDFromPath(path string) (string, error) {
	r := regexp.MustCompile(`d/(?P<deviceId>.*)/s.*`)
	m := r.FindStringSubmatch(path)
	var deviceID string
	if len(m) == 2 {
		deviceID = hex.EncodeToString([]byte(m[1]))
		return deviceID, nil
	}
	return "", errors.New("Device ID not found")
}

func getStateSubpath(path string) string {
	parts := strings.Split(path, "/s")
	if len(parts) == 2 {
		subpath := parts[1]
		if strings.HasPrefix(subpath, "/") {
			return subpath[1:]
		}
		return subpath
	}
	return ""
}

func (cg *CoAPGateway) handlePostState(ctx context.Context, w mux.ResponseWriter, req *mux.Message) {
	path, _ := req.Options.Path()
	deviceID, err := getDeviceIDFromPath(path)
	if err != nil {
		err = w.SetResponse(codes.BadRequest, message.TextPlain, nil)
		if err != nil {
			cg.logger.Errorf("Device ID err: %v ", err)
		}
	}

	subpath := getStateSubpath(path)

	if req.Body == nil {
		err = w.SetResponse(codes.BadRequest, message.TextPlain, nil)
		if err != nil {
			cg.logger.Errorf("cannot set response: %v", err)
		}
		return
	}

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		cg.logger.Warnf("cannot read response: %v", err)
		err = w.SetResponse(codes.BadRequest, message.TextPlain, nil)
		if err != nil {
			cg.logger.Errorf("cannot set response: %v", err)
		}
		return
	}

	parsedData := make(map[string]interface{})
	format, err := req.Options.ContentFormat()
	if err != nil {
		format = message.TextPlain
	}

	defer func() {
		ctx, err := tag.New(ctx, tag.Insert(KeyFormat, format.String()))
		if err != nil {
			cg.logger.Errorf("err creating metric for request %v \n", err)
		}
		stats.Record(ctx, MMessageBytes.M(int64(len(data)+len(path))))
	}()

	if format == message.AppCBOR {
		v := make(map[string]interface{})
		err = cbor.Unmarshal(data, &v)
		cg.logger.Infof("Parsed Cbor %v", v)
		if subpath != "" {
			parsedData[subpath] = v
		} else {
			parsedData = v
		}
	} else {
		parsedData[subpath] = string(data)
	}

	updates, err := flatten.Flatten(parsedData, "", flatten.PathStyle)

	if err != nil {
		cg.logger.Infof("cannot flatten data: %v", err)
		err = w.SetResponse(codes.BadGateway, message.TextPlain, nil)
		return
	}

	fullUpdate, err := flat.Unflatten(updates, &flat.Options{
		Delimiter: "/",
	})

	if err != nil {
		cg.logger.Errorf("cannot unflatten data: %v", err)
		err = w.SetResponse(codes.BadGateway, message.TextPlain, nil)
		return
	}

	updatesBody, err := json.Marshal(fullUpdate)
	if err == nil {
		err = cg.dataTopic.Send(req.Context, &pubsub.Message{
			Body: updatesBody,
			Metadata: map[string]string{
				"deviceID": deviceID,
				"time":     time.Now().String(),
			},
		})
		if err != nil {
			cg.logger.Errorf("Err publishing to message router: %v\n", err)
		} else {
			cg.logger.Infof("Message sent to router \n")
		}
	}

	cg.logger.Infof("Payload for devID %s - path %s - subpath %s, %v", deviceID, path, subpath, updates)
	err = w.SetResponse(codes.Valid, message.TextPlain, bytes.NewReader([]byte("OK")))
	if err != nil {
		cg.logger.Errorf("cannot set response: %v", err)
	}
}

func (cg *CoAPGateway) Start() {
	cg.router.Use(cg.routerMiddleware)
	cg.router.Use(cg.registerClient)

	registerMetrics()

	cg.logger.Info("Starting CoAP Gateway...")
	if cg.port > 0 {
		go func() {
			cg.logger.Fatalf("Error starting listener : %v",
				coap.ListenAndServe(
					"udp",
					":"+strconv.Itoa(cg.port),
					cg.router,
				))
		}()
	}

	if cg.tlsPort > 0 {

		certificate := util.GetCert()
		root, err := util.GetRootCert()
		if err != nil {
			cg.logger.Fatalf("err opening server cert: %v", err)
		}

		certPool := x509.NewCertPool()
		cert, err := x509.ParseCertificate(root.Certificate[0])
		if err != nil {
			cg.logger.Fatalf("err parsing server cert: %v", err)
		}
		certPool.AddCert(cert)

		go func() {
			cg.logger.Fatalf("Error starting dtls listener : %v",
				coap.ListenAndServeDTLS(
					"udp",
					":"+strconv.Itoa(cg.tlsPort),
					&dtls.Config{
						Certificates:         []tls.Certificate{*certificate},
						ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
						ClientAuth:           dtls.RequireAndVerifyClientCert,
						ClientCAs:            certPool,
					},
					cg.router,
				))
		}()
	}
}
