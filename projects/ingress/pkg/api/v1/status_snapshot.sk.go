// Code generated by solo-kit. DO NOT EDIT.

package v1

import (
	"fmt"

	"github.com/solo-io/go-utils/hashutils"
	"go.uber.org/zap"
)

type StatusSnapshot struct {
	Services  KubeServiceList
	Ingresses IngressList
}

func (s StatusSnapshot) Clone() StatusSnapshot {
	return StatusSnapshot{
		Services:  s.Services.Clone(),
		Ingresses: s.Ingresses.Clone(),
	}
}

func (s StatusSnapshot) Hash() uint64 {
	return hashutils.HashAll(
		s.hashServices(),
		s.hashIngresses(),
	)
}

func (s StatusSnapshot) hashServices() uint64 {
	return hashutils.HashAll(s.Services.AsInterfaces()...)
}

func (s StatusSnapshot) hashIngresses() uint64 {
	return hashutils.HashAll(s.Ingresses.AsInterfaces()...)
}

func (s StatusSnapshot) HashFields() []zap.Field {
	var fields []zap.Field
	fields = append(fields, zap.Uint64("services", s.hashServices()))
	fields = append(fields, zap.Uint64("ingresses", s.hashIngresses()))

	return append(fields, zap.Uint64("snapshotHash", s.Hash()))
}

type StatusSnapshotStringer struct {
	Version   uint64
	Services  []string
	Ingresses []string
}

func (ss StatusSnapshotStringer) String() string {
	s := fmt.Sprintf("StatusSnapshot %v\n", ss.Version)

	s += fmt.Sprintf("  Services %v\n", len(ss.Services))
	for _, name := range ss.Services {
		s += fmt.Sprintf("    %v\n", name)
	}

	s += fmt.Sprintf("  Ingresses %v\n", len(ss.Ingresses))
	for _, name := range ss.Ingresses {
		s += fmt.Sprintf("    %v\n", name)
	}

	return s
}

func (s StatusSnapshot) Stringer() StatusSnapshotStringer {
	return StatusSnapshotStringer{
		Version:   s.Hash(),
		Services:  s.Services.NamespacesDotNames(),
		Ingresses: s.Ingresses.NamespacesDotNames(),
	}
}
