package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"

	"github.com/skycoin/skycoin/src/api/webrpc"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/coin"
	"github.com/skycoin/skycoin/src/daemon"
	"github.com/skycoin/skycoin/src/gui"
	"github.com/skycoin/skycoin/src/util/browser"
	"github.com/skycoin/skycoin/src/util/cert"
	"github.com/skycoin/skycoin/src/util/file"
	"github.com/skycoin/skycoin/src/util/logging"
	"github.com/skycoin/skycoin/src/visor"
)

var (
	// Version node version which will be set when build wallet by LDFLAGS
	Version = "0.21.1"
	// Commit id
	Commit = ""

	help = false

	logger     = logging.MustGetLogger("main")
	logFormat  = "[suncoin.%{module}:%{level}] %{message}"
	logModules = []string{
		"main",
		"daemon",
		"coin",
		"gui",
		"util",
		"visor",
		"wallet",
		"gnet",
		"pex",
		"webrpc",
	}

	//TODO: Move time and other genesis block settigns from visor, to here
	GenesisSignatureStr = "3a2c8762df667edb5aa0cda6db52c36d490951bb35ff27ade65e76963f2bb7170be73e85474c45784cab2acbd9dbe1853d073e954badf8a395e9db7cb3261d1700"
	GenesisAddressStr   = "5L1jvbwtGS8eL3afA2gsqTBc8KEPFDDRjZ"
	BlockchainPubkeyStr = "0255434580f86e14a26e1d5c59b0626dfa28003741c475155aeedaa92af797d043"
	BlockchainSeckeyStr = ""

	GenesisTimestamp  uint64 = 1494861716
	GenesisCoinVolume uint64 = 300e12

	//GenesisTimestamp: 1426562704,
	//GenesisCoinVolume: 100e12, //100e6 * 10e6

	DefaultConnections = []string{
		"116.62.220.158:7200",
		"119.23.23.184:7200",
	}
)

// Config records the node's configuration
type Config struct {
	// Disable peer exchange
	DisablePEX bool
	// Download peer list
	DownloadPeerList bool
	// Download the peers list from this URL
	PeerListURL string
	// Don't make any outgoing connections
	DisableOutgoingConnections bool
	// Don't allowing incoming connections
	DisableIncomingConnections bool
	// Disables networking altogether
	DisableNetworking bool
	// Disables wallet API
	DisableWalletApi bool
	// Disable CSRF check in the wallet api
	DisableCSRF bool

	// Only run on localhost and only connect to others on localhost
	LocalhostOnly bool
	// Which address to serve on. Leave blank to automatically assign to a
	// public interface
	Address string
	//gnet uses this for TCP incoming and outgoing
	Port int
	//max outgoing connections to maintain
	MaxOutgoingConnections int
	// How often to make outgoing connections
	OutgoingConnectionsRate time.Duration
	// PeerlistSize represents the maximum number of peers that the pex would maintain
	PeerlistSize int
	// Wallet Address Version
	//AddressVersion string
	// Remote web interface
	WebInterface      bool
	WebInterfacePort  int
	WebInterfaceAddr  string
	WebInterfaceCert  string
	WebInterfaceKey   string
	WebInterfaceHTTPS bool

	RPCInterface     bool
	RPCInterfacePort int
	RPCInterfaceAddr string

	// Launch System Default Browser after client startup
	LaunchBrowser bool

	// If true, print the configured client web interface address and exit
	PrintWebInterfaceAddress bool

	// Data directory holds app data -- defaults to ~/.suncoin
	DataDirectory string
	// GUI directory contains assets for the html gui
	GUIDirectory string
	// Logging
	ColorLog bool
	// This is the value registered with flag, it is converted to LogLevel after parsing
	LogLevel        string
	DisablePingPong bool
	// Wallets
	// Defaults to ${DataDirectory}/wallets/
	WalletDirectory string

	RunMaster bool

	GenesisSignature cipher.Sig
	GenesisTimestamp uint64
	GenesisAddress   cipher.Address

	BlockchainPubkey cipher.PubKey
	BlockchainSeckey cipher.SecKey

	/* Developer options */

	// Enable cpu profiling
	ProfileCPU bool
	// Where the file is written to
	ProfileCPUFile string
	// HTTP profiling interface (see http://golang.org/pkg/net/http/pprof/)
	HTTPProf bool
	// Will force it to connect to this ip:port, instead of waiting for it
	// to show up as a peer
	ConnectTo string

	DBPath       string
	Arbitrating  bool
	RPCThreadNum uint // rpc number
	Logtofile    bool
	Logtogui     bool
	LogBuffSize  int
}

func (c *Config) register() {
	flag.BoolVar(&help, "help", false, "Show help")
	flag.BoolVar(&c.DisablePEX, "disable-pex", c.DisablePEX, "disable PEX peer discovery")
	flag.BoolVar(&c.DownloadPeerList, "download-peerlist", c.DownloadPeerList, "download a peers.txt from -peerlist-url")
	flag.StringVar(&c.PeerListURL, "peerlist-url", c.PeerListURL, "with -download-peerlist=true, download a peers.txt file from this url")
	flag.BoolVar(&c.DisableOutgoingConnections, "disable-outgoing", c.DisableOutgoingConnections, "Don't make outgoing connections")
	flag.BoolVar(&c.DisableIncomingConnections, "disable-incoming", c.DisableIncomingConnections, "Don't make incoming connections")
	flag.BoolVar(&c.DisableNetworking, "disable-networking", c.DisableNetworking, "Disable all network activity")
	flag.BoolVar(&c.DisableWalletApi, "disable-wallet-api", c.DisableWalletApi, "Disable the wallet API")
	flag.BoolVar(&c.DisableCSRF, "disable-csrf", c.DisableCSRF, "disable csrf check")
	flag.StringVar(&c.Address, "address", c.Address, "IP Address to run application on. Leave empty to default to a public interface")
	flag.IntVar(&c.Port, "port", c.Port, "Port to run application on")

	flag.BoolVar(&c.WebInterface, "web-interface", c.WebInterface, "enable the web interface")
	flag.IntVar(&c.WebInterfacePort, "web-interface-port", c.WebInterfacePort, "port to serve web interface on")
	flag.StringVar(&c.WebInterfaceAddr, "web-interface-addr", c.WebInterfaceAddr, "addr to serve web interface on")
	flag.StringVar(&c.WebInterfaceCert, "web-interface-cert", c.WebInterfaceCert, "cert.pem file for web interface HTTPS. If not provided, will use cert.pem in -data-directory")
	flag.StringVar(&c.WebInterfaceKey, "web-interface-key", c.WebInterfaceKey, "key.pem file for web interface HTTPS. If not provided, will use key.pem in -data-directory")
	flag.BoolVar(&c.WebInterfaceHTTPS, "web-interface-https", c.WebInterfaceHTTPS, "enable HTTPS for web interface")

	flag.BoolVar(&c.RPCInterface, "rpc-interface", c.RPCInterface, "enable the rpc interface")
	flag.IntVar(&c.RPCInterfacePort, "rpc-interface-port", c.RPCInterfacePort, "port to serve rpc interface on")
	flag.StringVar(&c.RPCInterfaceAddr, "rpc-interface-addr", c.RPCInterfaceAddr, "addr to serve rpc interface on")
	flag.UintVar(&c.RPCThreadNum, "rpc-thread-num", 5, "rpc thread number")

	flag.BoolVar(&c.LaunchBrowser, "launch-browser", c.LaunchBrowser, "launch system default webbrowser at client startup")
	flag.BoolVar(&c.PrintWebInterfaceAddress, "print-web-interface-address", c.PrintWebInterfaceAddress, "print configured web interface address and exit")
	flag.StringVar(&c.DataDirectory, "data-dir", c.DataDirectory, "directory to store app data (defaults to ~/.skycoin)")
	flag.StringVar(&c.DBPath, "db-path", c.DBPath, "path of database file (defaults to ~/.skycoin/data.db)")
	flag.StringVar(&c.ConnectTo, "connect-to", c.ConnectTo, "connect to this ip only")
	flag.BoolVar(&c.ProfileCPU, "profile-cpu", c.ProfileCPU, "enable cpu profiling")
	flag.StringVar(&c.ProfileCPUFile, "profile-cpu-file", c.ProfileCPUFile, "where to write the cpu profile file")
	flag.BoolVar(&c.HTTPProf, "http-prof", c.HTTPProf, "Run the http profiling interface")
	flag.StringVar(&c.LogLevel, "log-level", c.LogLevel, "Choices are: debug, info, notice, warning, error, critical")
	flag.BoolVar(&c.ColorLog, "color-log", c.ColorLog, "Add terminal colors to log output")
	flag.BoolVar(&c.DisablePingPong, "no-ping-log", false, `disable "reply to ping" and "received pong" log messages`)
	flag.BoolVar(&c.Logtofile, "logtofile", false, "log to file")
	flag.StringVar(&c.GUIDirectory, "gui-dir", c.GUIDirectory, "static content directory for the html gui")

	// Key Configuration Data
	flag.BoolVar(&c.RunMaster, "master", c.RunMaster, "run the daemon as blockchain master server")

	flag.StringVar(&BlockchainPubkeyStr, "master-public-key", BlockchainPubkeyStr, "public key of the master chain")
	flag.StringVar(&BlockchainSeckeyStr, "master-secret-key", BlockchainSeckeyStr, "secret key, set for master")

	flag.StringVar(&GenesisAddressStr, "genesis-address", GenesisAddressStr, "genesis address")
	flag.StringVar(&GenesisSignatureStr, "genesis-signature", GenesisSignatureStr, "genesis block signature")
	flag.Uint64Var(&c.GenesisTimestamp, "genesis-timestamp", c.GenesisTimestamp, "genesis block timestamp")

	flag.StringVar(&c.WalletDirectory, "wallet-dir", c.WalletDirectory, "location of the wallet files. Defaults to ~/.skycoin/wallet/")
	flag.IntVar(&c.MaxOutgoingConnections, "max-outgoing-connections", 16, "The maximum outgoing connections allowed")
	flag.IntVar(&c.PeerlistSize, "peerlist-size", 65535, "The peer list size")
	flag.DurationVar(&c.OutgoingConnectionsRate, "connection-rate", c.OutgoingConnectionsRate, "How often to make an outgoing connection")
	flag.BoolVar(&c.LocalhostOnly, "localhost-only", c.LocalhostOnly, "Run on localhost and only connect to localhost peers")
	flag.BoolVar(&c.Arbitrating, "arbitrating", c.Arbitrating, "Run node in arbitrating mode")
	flag.BoolVar(&c.Logtogui, "logtogui", false, "log to gui")
	flag.IntVar(&c.LogBuffSize, "logbufsize", c.LogBuffSize, "Log size saved in memeory for gui show")
}

var devConfig Config = Config{
	// Disable peer exchange
	DisablePEX: false,
	// Don't make any outgoing connections
	DisableOutgoingConnections: false,
	// Don't allowing incoming connections
	DisableIncomingConnections: false,
	// Disables networking altogether
	DisableNetworking: false,
	// Disable wallet API
	DisableWalletApi: false,
	// Disable CSRF check in the wallet api
	DisableCSRF: false,
	// Only run on localhost and only connect to others on localhost
	LocalhostOnly: false,
	// Which address to serve on. Leave blank to automatically assign to a
	// public interface
	Address: "",
	//gnet uses this for TCP incoming and outgoing
	Port: 7200,
	// MaxOutgoingConnections is the maximum outgoing connections allowed.
	MaxOutgoingConnections: 16,
	DownloadPeerList:       false,
	PeerListURL:            "",
	// How often to make outgoing connections, in seconds
	OutgoingConnectionsRate: time.Second * 5,
	PeerlistSize:            65535,
	// Wallet Address Version
	//AddressVersion: "test",
	// Remote web interface
	WebInterface:             true,
	WebInterfacePort:         7620,
	WebInterfaceAddr:         "127.0.0.1",
	WebInterfaceCert:         "",
	WebInterfaceKey:          "",
	WebInterfaceHTTPS:        false,
	PrintWebInterfaceAddress: false,

	RPCInterface:     true,
	RPCInterfacePort: 7630,
	RPCInterfaceAddr: "127.0.0.1",
	RPCThreadNum:     5,

	LaunchBrowser: true,
	// Data directory holds app data -- defaults to ~/.suncoin
	DataDirectory: ".suncoin",
	// Web GUI static resources
	GUIDirectory: "./src/gui/static/",
	// Logging
	ColorLog: true,
	LogLevel: "DEBUG",

	// Wallets
	WalletDirectory: "",

	// Centralized network configuration
	RunMaster:        false,
	BlockchainPubkey: cipher.PubKey{},
	BlockchainSeckey: cipher.SecKey{},

	GenesisAddress:   cipher.Address{},
	GenesisTimestamp: GenesisTimestamp,
	GenesisSignature: cipher.Sig{},

	/* Developer options */

	// Enable cpu profiling
	ProfileCPU: false,
	// Where the file is written to
	ProfileCPUFile: "suncoin.prof",
	// HTTP profiling interface (see http://golang.org/pkg/net/http/pprof/)
	HTTPProf: false,
	// Will force it to connect to this ip:port, instead of waiting for it
	// to show up as a peer
	ConnectTo:   "",
	LogBuffSize: 8388608, //1024*1024*8
}

func (c *Config) Parse() {
	c.register()
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}
	c.postProcess()
}

func (c *Config) postProcess() {
	var err error
	if GenesisSignatureStr != "" {
		c.GenesisSignature, err = cipher.SigFromHex(GenesisSignatureStr)
		panicIfError(err, "Invalid Signature")
	}
	if GenesisAddressStr != "" {
		c.GenesisAddress, err = cipher.DecodeBase58Address(GenesisAddressStr)
		panicIfError(err, "Invalid Address")
	}
	if BlockchainPubkeyStr != "" {
		c.BlockchainPubkey, err = cipher.PubKeyFromHex(BlockchainPubkeyStr)
		panicIfError(err, "Invalid Pubkey")
	}
	if BlockchainSeckeyStr != "" {
		c.BlockchainSeckey, err = cipher.SecKeyFromHex(BlockchainSeckeyStr)
		panicIfError(err, "Invalid Seckey")
		BlockchainSeckeyStr = ""
	}
	if BlockchainSeckeyStr != "" {
		c.BlockchainSeckey = cipher.SecKey{}
	}

	c.DataDirectory, err = file.InitDataDir(c.DataDirectory)
	panicIfError(err, "Invalid DataDirectory")

	if c.WebInterfaceCert == "" {
		c.WebInterfaceCert = filepath.Join(c.DataDirectory, "cert.pem")
	}
	if c.WebInterfaceKey == "" {
		c.WebInterfaceKey = filepath.Join(c.DataDirectory, "key.pem")
	}

	if c.WalletDirectory == "" {
		c.WalletDirectory = filepath.Join(c.DataDirectory, "wallets/")
	}

	if c.DBPath == "" {
		c.DBPath = filepath.Join(c.DataDirectory, "data.db")
	}

	if c.RunMaster {
		// Run in arbitrating mode if the node is master
		c.Arbitrating = true
	}
}

func panicIfError(err error, msg string, args ...interface{}) {
	if err != nil {
		log.Panicf(msg+": %v", append(args, err)...)
	}
}

func printProgramStatus() {
	p := pprof.Lookup("goroutine")
	if err := p.WriteTo(os.Stdout, 2); err != nil {
		fmt.Println("ERROR:", err)
		return
	}
}

// Catches SIGUSR1 and prints internal program state
func catchDebug() {
	sigchan := make(chan os.Signal, 1)
	//signal.Notify(sigchan, syscall.SIGUSR1)
	signal.Notify(sigchan, syscall.Signal(0xa)) // SIGUSR1 = Signal(0xa)
	for {
		select {
		case <-sigchan:
			printProgramStatus()
		}
	}
}

func catchInterrupt(quit chan<- struct{}) {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
	signal.Stop(sigchan)
	close(quit)

	// If ctrl-c is called again, panic so that the program state can be examined.
	// Ctrl-c would be called again if program shutdown was stuck.
	go catchInterruptPanic()
}

// catchInterruptPanic catches os.Interrupt and panics
func catchInterruptPanic() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt)
	<-sigchan
	signal.Stop(sigchan)
	printProgramStatus()
	panic("SIGINT")
}

func createGUI(c *Config, d *daemon.Daemon, host string, quit chan struct{}) (*gui.Server, error) {
	var s *gui.Server
	var err error

	config := gui.ServerConfig{
		StaticDir:   c.GUIDirectory,
		DisableCSRF: c.DisableCSRF,
	}

	if c.WebInterfaceHTTPS {
		// Verify cert/key parameters, and if neither exist, create them
		if err := cert.CreateCertIfNotExists(host, c.WebInterfaceCert, c.WebInterfaceKey, "Skycoind"); err != nil {
			logger.Error("gui.CreateCertIfNotExists failure: %v", err)
			return nil, err
		}

		s, err = gui.CreateHTTPS(host, config, d, c.WebInterfaceCert, c.WebInterfaceKey)
	} else {
		s, err = gui.Create(host, config, d)
	}
	if err != nil {
		logger.Error("Failed to start web GUI: %v", err)
		return nil, err
	}

	return s, nil
}

// init logging settings
func initLogging(dataDir string, level string, color, logtofile bool) (func(), error) {
	logCfg := logging.DevLogConfig(logModules)
	logCfg.Format = logFormat
	logCfg.Colors = color
	logCfg.Level = level

	var fd *os.File
	if logtofile {
		logDir := filepath.Join(dataDir, "logs")
		if err := createDirIfNotExist(logDir); err != nil {
			log.Println("initial logs folder failed", err)
			return nil, fmt.Errorf("init log folder fail, %v", err)
		}

		// open log file
		tf := "2006-01-02-030405"
		logfile := filepath.Join(logDir,
			fmt.Sprintf("%s-v%s.log", time.Now().Format(tf), Version))
		var err error
		fd, err = os.OpenFile(logfile, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			return nil, err
		}

		logCfg.Output = io.MultiWriter(os.Stdout, fd)
	}

	logCfg.InitLogger()

	return func() {
		logger.Info("Log file closed")
		if fd != nil {
			fd.Close()
		}
	}, nil
}

func initProfiling(httpProf, profileCPU bool, profileCPUFile string) {
	if profileCPU {
		f, err := os.Create(profileCPUFile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if httpProf {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}
}

func configureDaemon(c *Config) daemon.Config {
	//cipher.SetAddressVersion(c.AddressVersion)

	dc := daemon.NewConfig()
	dc.Pex.DataDirectory = c.DataDirectory
	dc.Pex.Disabled = c.DisablePEX
	dc.Pex.Port = c.Port
	dc.Pex.Max = c.PeerlistSize
	dc.Pex.DownloadPeerList = c.DownloadPeerList
	dc.Pex.PeerListURL = c.PeerListURL
	dc.Daemon.DisableOutgoingConnections = c.DisableOutgoingConnections
	dc.Daemon.DisableIncomingConnections = c.DisableIncomingConnections
	dc.Daemon.DisableNetworking = c.DisableNetworking
	dc.Daemon.Port = c.Port
	dc.Daemon.Address = c.Address
	dc.Daemon.LocalhostOnly = c.LocalhostOnly
	dc.Daemon.OutgoingMax = c.MaxOutgoingConnections
	dc.Daemon.DataDirectory = c.DataDirectory
	dc.Daemon.LogPings = !c.DisablePingPong

	if c.OutgoingConnectionsRate == 0 {
		c.OutgoingConnectionsRate = time.Millisecond
	}
	dc.Daemon.OutgoingRate = c.OutgoingConnectionsRate
	dc.Visor.Config.IsMaster = c.RunMaster

	dc.Visor.Config.BlockchainPubkey = c.BlockchainPubkey
	dc.Visor.Config.BlockchainSeckey = c.BlockchainSeckey

	dc.Visor.Config.GenesisAddress = c.GenesisAddress
	dc.Visor.Config.GenesisSignature = c.GenesisSignature
	dc.Visor.Config.GenesisTimestamp = c.GenesisTimestamp
	dc.Visor.Config.GenesisCoinVolume = GenesisCoinVolume
	dc.Visor.Config.DBPath = c.DBPath
	dc.Visor.Config.Arbitrating = c.Arbitrating
	dc.Visor.Config.WalletDirectory = c.WalletDirectory
	dc.Visor.Config.BuildInfo = visor.BuildInfo{
		Version: Version,
		Commit:  Commit,
	}

	dc.Gateway.DisableWalletAPI = c.DisableWalletApi

	return dc
}

// Run starts the suncoin node
func Run(c *Config) {
	defer func() {
		// try catch panic in main thread
		if r := recover(); r != nil {
			logger.Error("recover: %v\nstack:%v", r, string(debug.Stack()))
		}
	}()

	c.GUIDirectory = file.ResolveResourceDirectory(c.GUIDirectory)

	scheme := "http"
	if c.WebInterfaceHTTPS {
		scheme = "https"
	}
	host := fmt.Sprintf("%s:%d", c.WebInterfaceAddr, c.WebInterfacePort)
	fullAddress := fmt.Sprintf("%s://%s", scheme, host)
	logger.Critical("Full address: %s", fullAddress)
	if c.PrintWebInterfaceAddress {
		fmt.Println(fullAddress)
	}

	initProfiling(c.HTTPProf, c.ProfileCPU, c.ProfileCPUFile)

	closelog, err := initLogging(c.DataDirectory, c.LogLevel, c.ColorLog, c.Logtofile)
	if err != nil {
		fmt.Println(err)
		return
	}

	var wg sync.WaitGroup

	// If the user Ctrl-C's, shutdown properly
	quit := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		catchInterrupt(quit)
	}()

	// Watch for SIGUSR1
	wg.Add(1)
	func() {
		defer wg.Done()
		go catchDebug()
	}()

	// creates blockchain instance
	dconf := configureDaemon(c)

	logger.Info("Opening database %s", dconf.Visor.Config.DBPath)
	db, err := visor.OpenDB(dconf.Visor.Config.DBPath)
	if err != nil {
		logger.Error("Database failed to open: %v. Is another suncoin instance running?", err)
		return
	}

	d, err := daemon.NewDaemon(dconf, db, DefaultConnections)
	if err != nil {
		logger.Error("%v", err)
		return
	}

	var rpc *webrpc.WebRPC
	if c.RPCInterface {
		rpcAddr := fmt.Sprintf("%v:%v", c.RPCInterfaceAddr, c.RPCInterfacePort)
		rpc, err = webrpc.New(rpcAddr, d.Gateway)
		if err != nil {
			logger.Error("%v", err)
			return
		}
		rpc.ChanBuffSize = 1000
		rpc.WorkerNum = c.RPCThreadNum
	}

	var webInterface *gui.Server
	if c.WebInterface {
		webInterface, err = createGUI(c, d, host, quit)
		if err != nil {
			logger.Error("%v", err)
			return
		}
	}

	// Debug only - forces connection on start.  Violates thread safety.
	if c.ConnectTo != "" {
		if err := d.Pool.Pool.Connect(c.ConnectTo); err != nil {
			logger.Error("Force connect %s failed, %v", c.ConnectTo, err)
			return
		}
	}

	errC := make(chan error, 10)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := d.Run(); err != nil {
			logger.Error("%v", err)
			errC <- err
		}
	}()

	// start the webrpc
	if c.RPCInterface {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := rpc.Run(); err != nil {
				logger.Error("%v", err)
				errC <- err
			}
		}()
	}

	if c.WebInterface {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := webInterface.Serve(); err != nil {
				logger.Error("%v", err)
				errC <- err
			}
		}()

		if c.LaunchBrowser {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Wait a moment just to make sure the http interface is up
				time.Sleep(time.Millisecond * 100)

				logger.Info("Launching System Browser with %s", fullAddress)
				if err := browser.Open(fullAddress); err != nil {
					logger.Error(err.Error())
					return
				}
			}()
		}
	}
	/*
	   time.Sleep(5)
	   tx := InitTransaction()
	   _ = tx
	   err, _ = d.Visor.Visor.InjectTransaction(tx)
	   if err != nil {
	       log.Panic(err)
	   }
	*/

	//first transaction
	// if c.RunMaster == true {
	// 	go func() {
	// 		for d.Visor.HeadBkSeq() < 2 {
	// 			time.Sleep(5 * time.Second)
	// 			tx := InitTransaction()
	// 			_, err := d.Visor.InjectTransaction(tx, d.Pool)
	// 			if err != nil {
	// 				log.Panic(err)
	// 			}
	// 			logger.Critical("Inject transaction success")
	// 		}
	// 	}()
	// }

	select {
	case <-quit:
	case err := <-errC:
		logger.Error("%v", err)
	}

	logger.Info("Shutting down...")
	if rpc != nil {
		rpc.Shutdown()
	}
	if webInterface != nil {
		webInterface.Shutdown()
	}
	d.Shutdown()
	closelog()
	wg.Wait()
	logger.Info("Goodbye")
}

func main() {
	devConfig.Parse()
	Run(&devConfig)
}

//addresses for storage of coins
var AddrList []string = []string{
	"2JEc8JFzN2TGFy3wqeoe6eru3vwgq45sVSR",
	"2TvPvWdA4zvaqpcwTfPLkgHGQtDAzdqQCb7",
	"79xKvR3NQ7h4KD4vNt2PBLDtoKFDk1gG43",
	"27CnhwPzuZV6zVh5Pe6JmM8HeFu35z3Jw8K",
	"2M3pKMyx4NvfUFbWVvkT9Yv1STknFbL6VJp",
	"agGeGte7zwoCKbQPQkd8L5dTKL49uLHYua",
	"2GfRVqmmui6nddJkjjkjQ2fMbJFsWxezNV6",
	"2Xvd17c6tVoJRfQ4npZWGjcz1LYrsqdYGXx",
	"2Wgk1ghWPpD8NhZQi1ALbEKt8aNX9Np3Vim",
	"2RDA7WebLA6unbyezKSNUoQvMDZZZWdQuzH",
	"JSTLJ4FNxVwuhJdhBEvTVkwFWnqAhaDzic",
	"2bcCTbYxByGXNAYNfSAVA759qsCXD15XApt",
	"2UjMumrXD4r9CZKxPdBJwU4VHsVtMDB4Lv3",
	"jRDhoRhcpmRHjgcF6Emn7Yuj3mHRGzm262",
	"25T8PTJyLf6K4QisCZnv9J2EGLaramLt7fR",
	"TN5nFx6j5xoHj1jL5YZykaqeQ9UXKxyNC7",
	"rXLbAYaGehJwzja7Gqkc3NHBi9hUmrAFDR",
	"2S2Bfa4kvfauj2vWhhfbNRKwNLx5euxUBz5",
	"5C1nVed6c9zqfSWDoHu7fmsEyUVXpyX8Cy",
	"ZQ9QZRU1jGWRrXjGZWLdsZuqsQDa8BWWKo",
	"YQtJPhqx6sAAQY6ePnsV1FzJm3vM9HC9Tz",
	"2scTNQnfyZaDHPAhJxqMufYNa6pzzgmFJB",
	"B9Whv9d9TGYC7AHi9QJnANa1f2dY3Jny1r",
	"2EnSJzmbNdKNW8BBHavyTkVtyt6Y7j25ETd",
	"MdDzJg7RGqffo87XpmEVvFGPHAnpr9YmeS",
	"6Pc7ibaQ4CHQLH4HygrgRY9dMcGtRVoqSs",
	"2ZHRzCmZvdQV5R1Fi2c2STAo82VkHqgmvuc",
	"2Xay2CH2usdPYRDwoqQusMPxDEAuS76aAm2",
	"dtEeRybvCVVLeXqbmi4tzzzD5AMYFeeH9A",
	"2SpbEq7LzbEFvZFAFqR7fXBH9aibD2B6iQM",
	"CE5aNvp4qcBeHgaSyF5rFh9k6MamdyfHna",
	"2cEnxLn6h2ojHGC5TRS2BSSy7fgfFodi8dA",
	"WnW3cnehTBAsVDZ7nY8sNb2NM6NQabMPpe",
	"2hFKt4uBBpaT2Qt4QDuWi3cX3rAqQKKwtba",
	"6PxJUUfxZCGhMNueFCsPhGeCNHyXmCmPsd",
	"2atTZmiLmu8oxabcHUYFvQ9KcxMSAtxSKnu",
	"CRjdXLQb4CFbXxcEw2ER42Z95EJamjHkeB",
	"21TaWiWTCZBnC5Mhr8FFGkcg37jdjS12GPc",
	"27sZ4KJbJtbhgiBgtzsNknRc7H7h8YwDNq5",
	"w4MD35w8PTeexgQvbDPMpMf1UhZUVGkdhD",
	"A6FFCRPe7BvgE8oy7o5dhnuSVfSo8vtnkB",
	"2UawChW9sj9EEaVyimore9sbov3fRzif66k",
	"8Eb9dhfj6aTJf6os4M7zmaH1Gy95fmufUD",
	"2GW2zRVkxUkyGGxb4jXvA3TJVX13eS4AjTU",
	"FfsRWPhMRoSMmRFcmMb1knQiAC8RZDRNnA",
	"i5fkXenkrwfBhQUpMLYExJt4T8HYm5Swor",
	"2Q2FJJKULZHjbYdP8Nx5C4cBpoX8nw2gh4a",
	"2PWe6GiM3oKExhXsPHSbwnf3A5fffMKPEE7",
	"nP3BsoFkpbQgYHtp8onsQTFh32VkFQWibB",
	"w2xGLPgGgyTkSVnRgXhyiGpugxCFyMAFpc",
	"2BR9QGdm5hMxRkr4C5M21DXjWvxB26WjHeX",
	"Nnw5SgEKuUmsHGWBSzashbf5D98AUouXUh",
	"YVj3NEsacjsM67iPfyMY59vPuecwnp6QWz",
	"25Dvmd5uBvmZNsTRVDEBsQ6JRVpqqe7gDNe",
	"22oySh75yDv9vtBYHqgfYydEV8atTBwAnpp",
	"3Lrz2bDGcT7DprDeMJZ2guEyivAJ2phvtG",
	"td595rUgWWzMsZcudMnTVYiW6BRvaghPKw",
	"2ADzbt8F186254xMG6DvyMCfbsXdeiLA2qn",
	"NCpnHjMN9ta94SzJLZoFqCq9jpNqsHmeRh",
	"2JkBfkkvypKCsJYRVgMHwuTupf5u6j21onr",
	"2AgYKqcGFru1pkFbheNUzWkvKUKjY859AEJ",
	"2fJ7F7vviEWVRFBRKHGQVN59YiSKS2KPg5X",
	"2HnBLgUiZ3s843ik9RqgnArSNPXdagP99M1",
	"7wGF8xvTyVuKMAEsUWzbNqrmEwp56PoMcG",
	"2HzvsMgGU7Pg3pkg13AFAmt93cw2gEWiPc8",
	"2MURz4XbBRKriEL9UV8ixxZA837Hmpgtaqc",
	"2F3cbdN2tf1nKS6h5nfzUeMnoj7uRkjLy8e",
	"t4hvfvVChVk6qVajCnoqTTGNP4FTftJytu",
	"2bJsY7zjFH6iq6Fs1fnyX2LpzB6rPXVPWXC",
	"2R3MVa4AeY3UoZF2nubYDuxTesU91kiD2jd",
	"joVSFdo4CaA19Y2jbJJZ5BJyitZqBnmX9M",
	"Rw68wydU8cE33YLnNYq2eDksARfChZYFEp",
	"dDuYLwRQ1yvt9iQetEKrUGhPivAxB9n5M1",
	"2jWneyxn84PByzXMRTE3hnCJtNxBEYC2ufv",
	"D9KZjrFN2etU2SHFBA2G7jDSddh6ZCy26M",
	"b8KG4qRuxkHt29mpP5LNUuwQwBZhUG6RJB",
	"2LfmBFMFAsNf2ZDyNEwoH8RWNHmxSbQvqSb",
	"ruKpSMdayPGyFtYG9EH2QTpcrmTvF4EjAV",
	"2iej2x3fEk4sraSXeShVu65KH2NUAvJFg9N",
	"qcu4nqYcX5rGC3EmoXU8q22vDcPUMpPYXY",
	"2YhrGpKUWXfHrtW6c9FKVKqYrxVHVBXGind",
	"2H5mmwbibJz9KxpwRAuZ4ezvTSDL8iNUQGH",
	"zeh9v9afW7a3Ji5e7SGdzb3Sk2xajTXUMf",
	"xV3XQzAR86Y6r7qA49BsFcLxpQ4xhtASq7",
	"EtNKuCeca61htJigtYuove3Yb6H2Scpitk",
	"kfzfkb7TBQLGaBSN4ssjyfMNfscYjdYcy5",
	"27AKWwt4pxtixUM9PAG4J1at7w8oBivJms5",
	"2F5HzZU4RNYZbjmbCCXbDAsHAcqmAMZkpkp",
	"bnEUiH3HVq95pySibLRxHrTLgxQJhsTgRi",
	"7n9m5HZrVDVCNmVYirkG6WB18fHRnF3ZwK",
	"2GpNJcvfoBLTd21oVFvGmvyshmu5GjUDNgA",
	"2LM8uSr35BvyXNihhzTdSNasbYVX4yEGuUi",
	"zhixmLqy3fYcRUAMZZZuABtwVnxwfXriV3",
	"9n5yUAhSwbGZMMhC7KFNCjZnqC918tnjoY",
	"2m6PiUioXSQKAQCqVj7T5fzrVMAi5DX8tU5",
	"2EvQ2MwPeqXqaHs9qRFzWAoByz4ph64jSKU",
	"KR6sPjZvE2KfpAEpmMCdqAvCrtASpNADHK",
	"rk973tk7wCJsRU9ExFHBKuPgvLeehjjRMy",
	"PLUcUcFNSnK2rXoP2Fd5ugqGCMzn1ksfM1",
	"KZULDtdtgSqhUzvVJLBhscFaXGHdVHZ9TU",
}

func InitTransaction() coin.Transaction {
	var tx coin.Transaction

	output := cipher.MustSHA256FromHex("7c2f4421e420e77380791488772f177362e5c6fb61bb46612bbe21b5eabc3050")
	tx.PushInput(output)

	for i := 0; i < 100; i++ {
		addr := cipher.MustDecodeBase58Address(AddrList[i])
		tx.PushOutput(addr, 3e12, 1) // 10e6*10e6
	}
	// seckeys := make([]cipher.SecKey, 1)
	// seckey := ""
	// seckeys[0] = cipher.MustSecKeyFromHex(seckey)
	// tx.SignInputs(seckeys)

	txs := make([]cipher.Sig, 1)
	sig := "f1f3342332af4b773513008079772b12feabe40dd7a81ee0b3449f8609995d8b55c9c8436921bcbf31fd6c485017c1ef6e251070bd93033abd7ea3825813bb8e00"
	txs[0] = cipher.MustSigFromHex(sig)
	tx.Sigs = txs

	tx.UpdateHeader()

	err := tx.Verify()

	if err != nil {
		log.Panic(err)
	}

	return tx
}

func createDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		return nil
	}

	return os.Mkdir(dir, 0777)
}
