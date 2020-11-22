package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dgraph-io/dgo/v200"
	"github.com/dgraph-io/dgo/v200/protos/api"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	_ "github.com/joho/godotenv/autoload"
	"github.com/kelseyhightower/envconfig"
	gjson "github.com/layeh/gopher-json"
	"github.com/machinebox/graphql"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/yuin/gopher-lua"
	"google.golang.org/grpc"
	"layeh.com/gopher-luar"
)

type maincfg struct {
	URL    string `default:"http://dgraph:8080"`
	DQL    string `default:"dgraph:9080"`
	SCRIPT string `default:"/app/init.lua"`
}

type Req struct {
	Resolver   string `json:"resolver"`
	AuthHeader struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"authHeader,omitempty"`
	Args    map[string]interface{}   `json:"args,omitempty"`
	Parents []map[string]interface{} `json:"parents,omitempty"`
}

var (
	L            *lua.LState
	client       *graphql.Client
	dgraphClient *dgo.Dgraph
	handlers     map[string]func(rq Req) []byte
)

func RegisterHandler(resolver string, handler func(Req) []byte) {
	if handlers == nil {
		handlers = make(map[string]func(Req) []byte)
	}
	handlers[resolver] = handler
}
func GetHandler(resolver string) func(rq Req) (result []byte) {
	return handlers[resolver]
}

func init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	r.Post("/graphql-worker", wrk)
	L = lua.NewState()
	defer L.Close()
	gjson.Preload(L)
	var mc maincfg
	envconfig.Process("dgraph", &mc)

	L.SetGlobal("graphql", L.NewFunction(CallGraphql))
	L.SetGlobal("dql", L.NewFunction(CallDGO))
	log.Info().Str("script", mc.SCRIPT).Msg("Loading lua script")
	if err := L.DoFile(mc.SCRIPT); err != nil {
		panic(err)
	}
	log.Info().Str("DGRAPH_URL", mc.URL).Msg("Using dgraph GraphQL")
	client = graphql.NewClient(mc.URL + "/graphql")
	log.Info().Str("DGRAPH_DQL", mc.DQL).Msg("Using dgraph DQL")

	conn, err := grpc.Dial(mc.DQL, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot connect to dgraph")
	}
	defer conn.Close()
	dgraphClient = dgo.NewDgraphClient(api.NewDgraphClient(conn))
	http.ListenAndServe(":8686", r)
}

func CallGraphql(L *lua.LState) int {
	gql := L.ToString(1) /* get argument */
	args := L.ToTable(2)
	log.Debug().Str("query", gql).Msg("Got query")
	req := graphql.NewRequest(gql)

	args.ForEach(func(k, v lua.LValue) {
		log.Debug().Str("k", k.String()).Str("v", v.String()).Msg("Set var")
		req.Var(k.String(), v)
	})
	req.Header.Set("Cache-Control", "no-cache")
	ctx := context.Background()
	var respData interface{}
	if err := client.Run(ctx, req, &respData); err != nil {
		log.Fatal().Err(err).Msg("Cannot query grapql")
	}
	L.Push(luar.New(L, respData))
	return 1 /* number of results */
}

func CallDGO(L *lua.LState) int {
	dql := L.ToString(1) /* get argument */
	args := L.ToTable(2)
	ctx := context.Background()
	variables := make(map[string]string)
	args.ForEach(func(k, v lua.LValue) {
		variables[k.String()] = v.String()
	})
	resp, err := dgraphClient.NewTxn().QueryWithVars(ctx, dql, variables)
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot query")
	}
	L.Push(luar.New(L, string(resp.Json)))
	return 1 /* number of results */
}

func wrk(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Error: %#s\n", err)
		panic(err)
	}
	var rq Req
	if err = json.Unmarshal(b, &rq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())))
		fmt.Printf("Error: %#s\n", err)
		return
	}
	h := GetHandler(rq.Resolver)
	if h != nil {
		ret := h(rq)
		w.Write([]byte(ret))
	} else {
		if err := L.CallByParam(lua.P{
			Fn:      L.GetGlobal("onRequest"),
			NRet:    1,
			Protect: true,
		}, luar.New(L, &rq)); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())))
			fmt.Printf("Error: %#s\n", err)
			return
		}
		ret := L.Get(-1) // returned value
		L.Pop(1)         // remove received value
		w.Write([]byte(ret.String()))
	}
}
