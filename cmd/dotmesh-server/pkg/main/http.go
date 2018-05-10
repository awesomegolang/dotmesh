package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	rpc "github.com/gorilla/rpc/v2"
	rpcjson "github.com/gorilla/rpc/v2/json2"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/openzipkin/zipkin-go-opentracing/examples/middleware"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	rpcRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rpc_requests_total",
			Help: "Number of rpc requests.",
		},
		[]string{"device"},
	)
)

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

// setting up and running our http server
// rpc and replication live in rpc.go and replication.go respectively

func (state *InMemoryState) runServer() {
	go func() {
		// for debugging:
		// http://stackoverflow.com/questions/19094099/how-to-dump-goroutine-stacktraces
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	r := rpc.NewServer()
	r.RegisterCodec(rpcjson.NewCodec(), "application/json")
	r.RegisterCodec(rpcjson.NewCodec(), "application/json;charset=UTF-8")
	d := NewDotmeshRPC(state)
	err := r.RegisterService(d, "") // deduces name from type name
	if err != nil {
		log.Printf("Error while registering services %s", err)
	}

	router := mux.NewRouter()

	// only use the zipkin middleware if we have a TRACE_ADDR
	if os.Getenv("TRACE_ADDR") != "" {
		tracer := opentracing.GlobalTracer()

		router.Handle("/rpc",
			middleware.FromHTTPRequest(tracer, "rpc")(NewAuthHandler(r)),
		)

		router.Handle(
			"/filesystems/{filesystem}/{fromSnap}/{toSnap}",
			middleware.FromHTTPRequest(tracer, "zfs-sender")(
				NewAuthHandler(state.NewZFSSendingServer()),
			),
		).Methods("GET")

		router.Handle(
			"/filesystems/{filesystem}/{fromSnap}/{toSnap}",
			middleware.FromHTTPRequest(tracer, "zfs-receiver")(
				NewAuthHandler(state.NewZFSReceivingServer()),
			),
		).Methods("POST")

	} else {
		router.Handle("/rpc", NewAuthHandler(r))

		router.Handle(
			"/filesystems/{filesystem}/{fromSnap}/{toSnap}",
			NewAuthHandler(state.NewZFSSendingServer()),
		).Methods("GET")

		router.Handle(
			"/filesystems/{filesystem}/{fromSnap}/{toSnap}",
			NewAuthHandler(state.NewZFSReceivingServer()),
		).Methods("POST")

	}

	router.HandleFunc("/status",
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "OK")
		},
	)

	router.Handle("/metrics", promhttp.Handler())

	if os.Getenv("PRINT_HTTP_LOGS") != "" {
		loggingRouter := handlers.LoggingHandler(getLogfile("requests"), router)
		err = http.ListenAndServe(fmt.Sprintf(":%s", SERVER_PORT), loggingRouter)
	} else {
		err = http.ListenAndServe(fmt.Sprintf(":%s", SERVER_PORT), router)
	}

	if err != nil {
		out(fmt.Sprintf("Unable to listen on port %s: '%s'\n", SERVER_PORT, err))
		log.Fatalf("Unable to listen on port %s: '%s'", SERVER_PORT, err)
	}
}

func (state *InMemoryState) runUnixDomainServer() {
	// if we have disabled flexvolume then we are not running inside Kubernetes
	// and do not need the unix domain socket
	if os.Getenv("DISABLE_FLEXVOLUME") != "" {
		return
	}
	r := rpc.NewServer()
	r.RegisterCodec(rpcjson.NewCodec(), "application/json")
	r.RegisterCodec(rpcjson.NewCodec(), "application/json;charset=UTF-8")
	d := NewDotmeshRPC(state)
	err := r.RegisterService(d, "") // deduces name from type name
	if err != nil {
		log.Printf("[runUnixDomainServer] Error while registering services %s", err)
	}

	// UNIX socket for flexvolume driver to talk to us
	FV_SOCKET := FLEXVOLUME_DIR + "/dm.sock"

	// Unlink any old socket lingering there
	if _, err := os.Stat(FV_SOCKET); err == nil {
		if err = os.Remove(FV_SOCKET); err != nil {
			log.Fatalf("[runUnixDomainServer] Could not clean up existing socket at %s: %v", FV_SOCKET, err)
		}
	}

	listener, err := net.Listen("unix", FV_SOCKET)
	if err != nil {
		log.Fatalf("[runUnixDomainServer] Could not listen on %s: %v", FV_SOCKET, err)
	}

	unixSocketRouter := mux.NewRouter()
	unixSocketRouter.Handle("/rpc", r)

	// pre-authenticated-as-admin rpc server for clever unix socket clients
	// only. intended for use by the flexvolume driver, hence the location on
	// disk.
	http.Serve(listener, NewAdminHandler(unixSocketRouter))
}

type AuthHandler struct {
	subHandler http.Handler
}

func auth(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	notAuth := func(w http.ResponseWriter) {
		http.Error(w, "Unauthorized.", 401)
	}
	// check for empty username, if so show a login box
	user, pass, _ := r.BasicAuth()
	if user == "" {
		notAuth(w)
		return r, fmt.Errorf("Permission denied.")
	}
	// ok, user has provided u/p, try to log them in
	authorized, passworded, err := CheckPassword(user, pass)
	if err != nil {
		log.Printf(
			"[AuthHandler] Error running check on %s: %s:",
			user, err,
		)
		http.Error(w, fmt.Sprintf("Error: %s.", err), 401)
		return r, err
	}
	if !authorized {
		notAuth(w)
		return r, fmt.Errorf("Permission denied.")
	}
	u, err := GetUserByName(user)
	if err != nil {
		log.Printf(
			"[AuthHandler] Unable to locate user %v: %v", user, err,
		)
		notAuth(w)
		return r, fmt.Errorf("Permission denied.")
	}
	r = r.WithContext(
		context.WithValue(context.WithValue(r.Context(), "authenticated-user-id", u.Id),
			"password-authenticated", passworded),
	)
	return r, nil
}

func (a AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r, err := auth(w, r)
	if err != nil {
		// Communicating the error upstream is handled by auth
		return
	}
	a.subHandler.ServeHTTP(w, r)
}

func NewAuthHandler(handler http.Handler) http.Handler {
	return AuthHandler{subHandler: handler}
}

func authHandlerFunc(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r, err := auth(w, r)
		if err != nil {
			return
		}
		f(w, r)
	}
}

// handler which makes all requests appear as the admin user!
// DANGER - only use for unix domain sockets.
func NewAdminHandler(handler http.Handler) http.Handler {
	return AdminHandler{subHandler: handler}
}

type AdminHandler struct {
	subHandler http.Handler
}

func (a AdminHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = r.WithContext(AdminContext(r.Context()))
	a.subHandler.ServeHTTP(w, r)
}
