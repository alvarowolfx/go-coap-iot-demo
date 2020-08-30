package coap

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	coap "github.com/go-ocf/go-coap/v2"
	"github.com/go-ocf/go-coap/v2/message"
	"github.com/go-ocf/go-coap/v2/message/codes"
	"github.com/go-ocf/go-coap/v2/mux"
	"github.com/jeremywohl/flatten"
	"github.com/nqd/flat"

	"github.com/fxamacker/cbor/v2"
	"gocloud.dev/pubsub"
)

type CoAPGateway struct {
	devices   sync.Map
	router    *mux.Router
	dataTopic *pubsub.Topic
}

func NewGateway(dataTopic *pubsub.Topic) *CoAPGateway {
	router := mux.NewRouter()
	return &CoAPGateway{
		router:    router,
		dataTopic: dataTopic,
	}
}

// Middleware function, which will be called for each request.
func (cg *CoAPGateway) routerMiddleware(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		path, err := r.Options.Path()
		if err == nil && strings.HasPrefix(path, "d/") {
			if r.Code == codes.POST {
				cg.handlePostState(w, r)
			}
		}
		next.ServeCOAP(w, r)
	})
}

func (cg *CoAPGateway) registerClient(next mux.Handler) mux.Handler {
	return mux.HandlerFunc(func(w mux.ResponseWriter, r *mux.Message) {
		log.Printf("Registering client %v \n", w.Client().RemoteAddr())
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
			log.Printf("subpath %s, later : %s", subpath, subpath[1:])
			return subpath[1:]
		}
		return subpath
	}
	return ""
}

func (cg *CoAPGateway) handlePostState(w mux.ResponseWriter, req *mux.Message) {
	path, _ := req.Options.Path()
	deviceID, err := getDeviceIDFromPath(path)
	if err != nil {
		err = w.SetResponse(codes.BadRequest, message.TextPlain, nil)
		if err != nil {
			log.Printf("Device ID err: %v", err)
		}
	}

	subpath := getStateSubpath(path)

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("cannot read response: %v", err)
		err = w.SetResponse(codes.BadRequest, message.TextPlain, nil)
		if err != nil {
			log.Printf("cannot set response: %v", err)
		}
		return
	}

	parsedData := make(map[string]interface{})
	format, err := req.Options.ContentFormat()
	if err != nil {
		format = message.TextPlain
	}

	if format == message.AppCBOR {
		v := make(map[string]interface{})
		err = cbor.Unmarshal(data, &v)
		log.Printf("Parsed Cbor %v", v)
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
		log.Printf("cannot flatten data: %v", err)
		err = w.SetResponse(codes.BadGateway, message.TextPlain, nil)
		return
	}

	fullUpdate, err := flat.Unflatten(updates, &flat.Options{
		Delimiter: "/",
	})
	if err != nil {
		log.Printf("cannot unflatten data: %v", err)
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
			log.Printf("Err publishing to message router: %v\n", err)
		} else {
			log.Printf("Message sent to router \n")
		}
	}

	log.Printf("Payload for devID %s - path %s - subpath %s, %v", deviceID, path, subpath, updates)
	err = w.SetResponse(codes.POST, message.TextPlain, bytes.NewReader([]byte("OK")))
	if err != nil {
		log.Printf("cannot set response: %v", err)
	}
}

func (cg *CoAPGateway) Start() {
	cg.router.Use(cg.routerMiddleware)
	cg.router.Use(cg.registerClient)

	log.Println("Starting CoAP Gateway...")
	log.Fatal(coap.ListenAndServe("udp", ":5683", cg.router))

	// for tcp
	// log.Fatal(coap.ListenAndServe("tcp", ":5688",  r))
	// for tcp-tls
	// log.Fatal(coap.ListenAndServeTLS("tcp", ":5688", &tls.Config{...}, r))
	// for udp-dtls
	// log.Fatal(coap.ListenAndServeDTLS("udp", ":5688", &dtls.Config{...}, r))
}
