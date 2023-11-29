package main

import (
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	. "github.com/dave/jennifer/jen"
	cp "github.com/otiai10/copy"
	"github.com/segmentio/ksuid"
	"go.dokari.do/internal/pkginfo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type require struct {
	path    string
	version string
}

var router require

func init() {
	router = require{
		path:    "github.com/go-chi/chi/v5",
		version: "v5.0.10",
	}
}

func main() {
	root, _ := filepath.Abs("example")
	log, sync := makeLogger()
	defer sync()

	log.Debug("parsing package info")
	pkg, err := pkginfo.ParsePkg(root)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("building application")
	dest, err := copyTempDest(root)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("generating main.go")
	f := NewFile("main")
	f.ImportName(router.path, "chi")
	f.Func().Id("main").Params().Block(
		Id("r").Op(":=").Qual(router.path, "NewRouter").Call(),
		Do(func(s *Statement) {
			for _, r := range pkg.Route {
				r.Gen(s.Id("r")).Line()
			}
		}),
		Qual("net/http", "ListenAndServe").Call(Lit(":3000"), Id("r")),
	)

	if err := f.Save(filepath.Join(dest, "main.go")); err != nil {
		log.Fatal(err)
	}

	modfile, err := pkginfo.FindModFile(dest)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("adding required dependencies")
	modfile.AddRequire(router.path, router.version)
	dat, err := modfile.Format()
	if err != nil {
		log.Fatal(err)
	}

	newMod := filepath.Join(dest, "go.mod")
	modStat, err := os.Stat(newMod)
	if err != nil {
		log.Fatal(err)
	}

	log.Debug("generating new go.sum file")
	os.WriteFile(newMod, dat, modStat.Mode())

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dest
	if _, err := cmd.Output(); err != nil {
		log.Fatal(err)
	}

	log.Debugf("starting server in %s", dest)
	go func() {
		cmd = exec.Command("go", "run", "main.go")
		cmd.Dir = dest
		log.Info("server running on http://localhost:3000")
		if _, err := cmd.Output(); err != nil {
			log.Fatal(err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Info("cleaning up")
	if err := os.RemoveAll(dest); err != nil {
		log.Fatal(err)
	}
}

func makeLogger() (*zap.SugaredLogger, func() error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := cfg.Build()
	log := logger.Sugar()
	logger.Sync()

	return log, logger.Sync
}

func copyTempDest(src string) (string, error) {
	dst := filepath.Join(os.TempDir(), "do-"+id())
	err := cp.Copy(src, dst)
	if err != nil {
		return "", err
	}

	return dst, nil
}

func id() string {
	return ksuid.New().String()
}
