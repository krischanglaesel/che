package exec

import (
	"github.com/eclipse/che/agents/go-agents/core/jsonrpc"
	"github.com/eclipse/che/agents/go-agents/core/jsonrpc/jsonrpctest"
	"github.com/eclipse/che/agents/go-agents/core/process"
	"testing"
	"log"
)

func TestJSONRPCStartProcess(t *testing.T) {
	jsonrpc.RegRoutesGroup(RPCRoutes)
	connRecorder := jsonrpctest.NewConnRecorder()
	channel := jsonrpc.NewChannel(connRecorder, jsonrpc.DefaultRouter)
	channel.Go()
	defer channel.Close()

	command := &process.Command{Name: "test", CommandLine: "echo test"}
	log.Println("BEFORE PUSH")
	connRecorder.PushNextReq(StartMethod, command)
	log.Println("AFTER PUSH")

	if err := connRecorder.WaitUntil(jsonrpctest.ReqSent(process.DiedEventType)); err != nil {
		t.Fatal(err)
	}

	connRecorder.GetAllRequest()
}
