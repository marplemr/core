package locator

import (
	"crypto/ecdsa"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/noxiouz/zapctx/ctxlog"
	"github.com/pkg/errors"
	pb "github.com/sonm-io/core/proto"
	"github.com/sonm-io/core/util"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

var errNodeNotFound = errors.New("node with given Eth address cannot be found")

type node struct {
	ethAddr common.Address
	ipAddr  []string
	ts      time.Time
}

type Locator struct {
	mx sync.Mutex

	conf        *LocatorConfig
	db          map[common.Address]*node
	ctx         context.Context
	ethKey      *ecdsa.PrivateKey
	grpc        *grpc.Server
	certRotator util.HitlessCertRotator
	creds       credentials.TransportCredentials
}

func (l *Locator) Announce(ctx context.Context, req *pb.AnnounceRequest) (*pb.Empty, error) {
	ethAddr, err := l.extractEthAddr(ctx)
	if err != nil {
		return nil, err
	}

	log.G(l.ctx).Info("handling Announce request",
		zap.Stringer("eth", ethAddr), zap.Strings("ips", req.IpAddr))

	l.putAnnounce(&node{
		ethAddr: ethAddr,
		ipAddr:  req.IpAddr,
	})

	return &pb.Empty{}, nil
}

func (l *Locator) Resolve(ctx context.Context, req *pb.ResolveRequest) (*pb.ResolveReply, error) {
	log.G(l.ctx).Info("handling Resolve request", zap.String("eth", req.EthAddr))

	if !common.IsHexAddress(req.EthAddr) {
		return nil, fmt.Errorf("invalid ethaddress %s", req.EthAddr)
	}

	n, err := l.getResolve(common.HexToAddress(req.EthAddr))
	if err != nil {
		return nil, err
	}

	return &pb.ResolveReply{IpAddr: n.ipAddr}, nil
}

func (l *Locator) Serve() error {
	lis, err := net.Listen("tcp", l.conf.ListenAddr)
	if err != nil {
		return err
	}

	return l.grpc.Serve(lis)
}

func (l *Locator) extractEthAddr(ctx context.Context) (common.Address, error) {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return common.Address{}, status.Error(codes.DataLoss, "failed to get peer from ctx")
	}

	switch info := pr.AuthInfo.(type) {
	case util.EthAuthInfo:
		return info.Wallet, nil
	default:
		return common.Address{}, status.Error(codes.Unauthenticated, "wrong AuthInfo type")
	}
}

func (l *Locator) putAnnounce(n *node) {
	l.mx.Lock()
	defer l.mx.Unlock()

	n.ts = time.Now()
	l.db[n.ethAddr] = n
}

func (l *Locator) getResolve(ethAddr common.Address) (*node, error) {
	l.mx.Lock()
	defer l.mx.Unlock()

	n, ok := l.db[ethAddr]
	if !ok {
		return nil, errNodeNotFound
	}

	return n, nil
}

func (l *Locator) cleanExpiredNodes() {
	t := time.NewTicker(l.conf.CleanupPeriod)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			l.traverseAndClean()
		}
	}
}

func (l *Locator) traverseAndClean() {
	deadline := time.Now().Add(-1 * l.conf.NodeTTL)

	l.mx.Lock()
	defer l.mx.Unlock()

	var (
		total = len(l.db)
		del   uint64
		keep  uint64
	)
	for addr, node := range l.db {
		if node.ts.Before(deadline) {
			delete(l.db, addr)
			del++
		} else {
			keep++
		}
	}

	log.G(l.ctx).Debug("expired nodes cleaned",
		zap.Int("total", total), zap.Uint64("keep", keep), zap.Uint64("del", del))
}

func NewLocator(ctx context.Context, conf *LocatorConfig, key *ecdsa.PrivateKey) (l *Locator, err error) {
	if key == nil {
		return nil, errors.Wrap(err, "private key should be provided")
	}

	l = &Locator{
		db:     make(map[common.Address]*node),
		conf:   conf,
		ctx:    ctx,
		ethKey: key,
	}

	var TLSConfig *tls.Config
	l.certRotator, TLSConfig, err = util.NewHitlessCertRotator(ctx, l.ethKey)
	if err != nil {
		return nil, err
	}

	l.creds = util.NewTLS(TLSConfig)
	srv := util.MakeGrpcServer(l.creds)
	l.grpc = srv

	go l.cleanExpiredNodes()

	pb.RegisterLocatorServer(srv, l)

	return l, nil
}
