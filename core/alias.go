package core

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"strings"

	"github.com/koron/nvgd/common_const"
	"github.com/koron/nvgd/ltsv"
	"github.com/koron/nvgd/resource"
)

type alias struct {
	from, to string
}

func (a alias) rewritePath(src string) string {
	if !strings.HasPrefix(src, "/") {
		return src
	}
	if !strings.HasPrefix(src[1:], a.to) {
		return src
	}
	return "/" + a.from + src[1+len(a.to):]
}

func (a alias) rewriteLTSV(src *resource.Resource) (*resource.Resource, error) {
	buf := &bytes.Buffer{}
	lr := ltsv.NewReader(src)
	for {
		s, err := lr.Read()
		if err != nil {
			src.Close()
			if !errors.Is(err, io.EOF) {
				return nil, err
			}
			break
		}
		for i, p := range s.Properties {
			s.Properties[i].Value = a.rewritePath(p.Value)
		}
		err = ltsv.Write(buf, s.Properties)
		if err != nil {
			return nil, err
		}
	}
	dst := resource.New(ioutil.NopCloser(buf))
	dst.Options = src.Options
	for _, k := range []string{common_const.UpLink, common_const.NextLink} {
		v, ok := dst.String(k)
		if !ok || v == "" {
			continue
		}
		dst.Options[k] = a.rewritePath(v)
	}
	return dst, nil
}

type aliases []alias

// apply aliases for compatibility with koron/night
var defaultAliases = aliases{
	{"files/", "file:///"},
	{"commands/", "command://"},
	{"config/", "config://"},
	{"help/", "help://"},
	{"version/", "version://"},
}

func (a aliases) apply(path string) (string, *alias) {
	for _, n := range a {
		if strings.HasPrefix(path, n.from) {
			return n.to + path[len(n.from):], &n
		}
	}
	return path, nil
}

func (a aliases) mergeMap(m map[string]string) aliases {
	dst := make(aliases, len(a), len(a)+len(m))
	copy(dst[:len(a)], a)
	if len(m) == 0 {
		return dst
	}
	for from, to := range m {
		dst = append(dst, alias{from: from, to: to})
	}
	return dst
}
