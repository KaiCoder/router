package server

import (
	"errors"
	"math/rand"
	"time"

	log "github.com/cihub/seelog"
)

type RmqConn struct {
	RmqInstNormal []Rmq_mgr
	RmqInstFailed []Rmq_mgr
}

