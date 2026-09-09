package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/taeels/enode/internal/enode"
	"github.com/taeels/enode/internal/panel"
)

// runPanelCmd 은 이 노드의 제어판을 띄운다 (enode panel).
//
// enodectl serve 가 이 하위명령으로 exec 위임한다 — 제어판의 net/http 표면을
// enodectl.exe 링크 그래프에서 떼어 심볼 상한을 지키기 위해서다 (decisions §2).
// 데몬(enode)은 원래 net/http 를 쓰므로 여기 붙어도 새로 무는 것이 없다.
func runPanelCmd(args []string) int {
	fs := flag.NewFlagSet("panel", flag.ContinueOnError)
	cfgPath := fs.String("config", "", "path to the node config file (also determines identity)")
	listen := fs.String("listen", "127.0.0.1:8081", "panel bind address; outside 127.0.0.1 requires panel_token")
	node := fs.String("node", "", "node name for display and log lookup")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	confPath, tried := enode.ResolveConfig(*cfgPath)
	if confPath == "" {
		log.Error("no config file found", "tried", strings.Join(tried, ", "))
		return 1
	}
	local, err := enode.LoadLocal(confPath)
	if err != nil {
		log.Error("cannot read config", "path", confPath, "err", err)
		return 1
	}
	// panel_token 은 정책 파일에 산다 (drain 과 같은 파일). 없으면 빈 값이고,
	// loopback 바인딩이면 New 가 받아들인다. LAN 이면 New 가 거부한다.
	pol, err := enode.ReadPolicyFile(confPath)
	if err != nil {
		log.Error("cannot read the policy file", "err", err)
		return 1
	}
	name := *node
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(confPath), filepath.Ext(confPath))
	}

	srv, err := panel.New(panel.Config{
		Node:         name,
		ConfigPath:   confPath,
		Listen:       *listen,
		MediatorBase: local.Mediator,
		Token:        local.Token,
		PanelToken:   pol.PanelToken,
	})
	if err != nil {
		log.Error("cannot start the panel", "err", err)
		return 1
	}
	log.Info("panel listening", "addr", *listen, "node", name, "config", confPath)
	if err := http.ListenAndServe(*listen, srv.Handler()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
