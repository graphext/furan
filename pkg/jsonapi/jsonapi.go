package jsonapi

import (
	"encoding/json"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/dollarshaveclub/furan/v2/pkg/generated/furanrpc"
	fgrpc "github.com/dollarshaveclub/furan/v2/pkg/grpc"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

// Handlers holds the handlers for the HTTP endpoints
type Handlers struct {
	gr      *fgrpc.Server
	LogFunc func(msg string, args ...interface{})
}

func NewHandlers(gr *fgrpc.Server) *Handlers {
	return &Handlers{
		gr: gr,
	}
}

var APIKeyHeader = "API-Key"

// addkey adds the API key from the incoming request (if present) to the context
func (h *Handlers) addkey(r *http.Request) context.Context {
	md := make(metadata.MD, 1)
	md[fgrpc.APIKeyLabel] = r.Header[APIKeyHeader]
	return metadata.NewIncomingContext(r.Context(), md)
}

func (h *Handlers) log(msg string, args ...interface{}) {
	if h.LogFunc != nil {
		h.LogFunc(msg, args...)
	}
}

func (h *Handlers) Register(r *mux.Router) {
	if r == nil {
		return
	}
	r.HandleFunc("/api/build", h.BuildRequestHandler).Methods("POST")
	r.HandleFunc("/api/build/{id}/status", h.BuildStatusHandler).Methods("GET")
	r.HandleFunc("/api/build/{id}/cancel", h.BuildCancelHandler).Methods("POST")
}

func (h *Handlers) handleRPCError(w http.ResponseWriter, err error) {
	w.Header().Add("Content-Type", "text/plain")
	switch status.Code(err) {
	case codes.InvalidArgument:
		w.WriteHeader(http.StatusBadRequest)
	case codes.Unauthenticated:
		w.WriteHeader(http.StatusUnauthorized)
	case codes.Internal:
		fallthrough
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Write([]byte(err.Error()))
}

// REST interface handlers (proxy to gRPC handlers)
func (h *Handlers) BuildRequestHandler(w http.ResponseWriter, r *http.Request) {
	req := furanrpc.BuildRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log("error unmarshaling request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := h.gr.StartBuild(h.addkey(r), &req)
	if err != nil {
		h.handleRPCError(w, err)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) BuildStatusHandler(w http.ResponseWriter, r *http.Request) {
	req := furanrpc.BuildStatusRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log("error unmarshaling request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := h.gr.GetBuildStatus(h.addkey(r), &req)
	if err != nil {
		h.handleRPCError(w, err)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) BuildCancelHandler(w http.ResponseWriter, r *http.Request) {
	req := furanrpc.BuildCancelRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log("error unmarshaling request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	resp, err := h.gr.CancelBuild(h.addkey(r), &req)
	if err != nil {
		h.handleRPCError(w, err)
		return
	}
	json.NewEncoder(w).Encode(resp)
}
