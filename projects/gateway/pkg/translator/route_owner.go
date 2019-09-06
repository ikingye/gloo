package translator

import (
	"encoding/json"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	v1 "github.com/solo-io/gloo/projects/gloo/pkg/api/v1"
	"github.com/solo-io/go-utils/protoutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
)

type RouteMetadata struct {
	Owners []OwnerRef `json:"owners"`
}

type OwnerRef struct {
	core.ResourceRef
	ResourceKind       string `json:"kind"`
	ObservedGeneration int64  `json:"observedGeneration"`
}

func RouteMetaFromStruct(s *types.Struct) (*RouteMetadata, error) {
	if s == nil {
		return nil, nil
	}
	var m RouteMetadata
	if err := protoutils.UnmarshalStruct(s, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func routeMetaToStruct(meta *RouteMetadata) (*types.Struct, error) {
	data, err := json.Marshal(meta)
	var pb types.Struct
	err = jsonpb.UnmarshalString(string(data), &pb)
	return &pb, err
}

func setRouteMeta(route *v1.Route, meta *RouteMetadata) error {
	metaStruct, err := routeMetaToStruct(meta)
	if err != nil {
		return err
	}
	route.RouteMetadata = metaStruct
	return nil
}

func getRouteMeta(route *v1.Route) (*RouteMetadata, error) {
	if route.RouteMetadata == nil {
		return &RouteMetadata{}, nil
	}
	return RouteMetaFromStruct(route.RouteMetadata)
}

func appendOwner(route *v1.Route, owner resources.InputResource) error {
	meta, err := getRouteMeta(route)
	if err != nil {
		return errors.Wrapf(err, "getting route metadata")
	}
	meta.Owners = append(meta.Owners, makeOwnerRef(owner))
	return setRouteMeta(route, meta)
}

func makeOwnerRef(owner resources.InputResource) OwnerRef {
	return OwnerRef{
		ResourceRef:        owner.GetMetadata().Ref(),
		ResourceKind:       resources.Kind(owner),
		ObservedGeneration: owner.GetMetadata().Generation,
	}
}
