package dlv

import (
	"path/filepath"

	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/service"
	"github.com/go-delve/delve/service/debugger"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/go-delve/delve/service/rpccommon"
)

func Launch(program string) (*rpc2.RPCClient, error) {
	listener, clientConn := service.ListenerPipe()
	defer listener.Close()

	server := rpccommon.NewServer(&service.Config{
		Listener:    listener,
		ProcessArgs: []string{program},
		APIVersion:  2,
		Debugger: debugger.Config{
			WorkingDir:     filepath.Dir(program),
			Backend:        "default",
			ExecuteKind:    debugger.ExecutingExistingFile,
			CheckGoVersion: true,
			Stdout: proc.OutputRedirect{
				Path: "/dev/null",
			},
			Stderr: proc.OutputRedirect{
				Path: "/dev/null",
			},
		},
	})
	if err := server.Run(); err != nil {
		return nil, err
	}

	client := rpc2.NewClientFromConn(clientConn)
	return client, nil
}
