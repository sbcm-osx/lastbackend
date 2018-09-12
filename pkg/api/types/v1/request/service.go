//
// Last.Backend LLC CONFIDENTIAL
// __________________
//
// [2014] - [2018] Last.Backend LLC
// All Rights Reserved.
//
// NOTICE:  All information contained herein is, and remains
// the property of Last.Backend LLC and its suppliers,
// if any.  The intellectual and technical concepts contained
// herein are proprietary to Last.Backend LLC
// and its suppliers and may be covered by Russian Federation and Foreign Patents,
// patents in process, and are protected by trade secret or copyright law.
// Dissemination of this information or reproduction of this material
// is strictly forbidden unless prior written permission is obtained
// from Last.Backend LLC.
//

package request

import (
	"encoding/json"
	"github.com/lastbackend/lastbackend/pkg/distribution/types"
	"gopkg.in/yaml.v2"
	"strconv"
	"strings"
	"time"
)

type ServiceManifest struct {
	Meta ServiceManifestMeta `json:"meta,omitempty" yaml:"meta,omitempty"`
	Spec ServiceManifestSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

type ServiceManifestMeta struct {
	RuntimeMeta `yaml:",inline"`
}

type ServiceManifestSpec struct {
	Selector *ManifestSpecSelector `json:"selector,omitempty" yaml:"selector,omitempty"`
	Replicas *int                  `json:"replicas,omitempty" yaml:"replicas,omitempty"`
	Network  *ManifestSpecNetwork  `json:"network,omitempty" yaml:"network,omitempty"`
	Strategy *ManifestSpecStrategy `json:"strategy,omitempty" yaml:"strategy,omitempty"`
	Template *ManifestSpecTemplate `json:"template,omitempty" yaml:"template,omitempty"`
}

func (s *ServiceManifest) FromJson(data []byte) error {
	return json.Unmarshal(data, s)
}

func (s *ServiceManifest) ToJson() ([]byte, error) {
	return json.Marshal(s)
}

func (s *ServiceManifest) FromYaml(data []byte) error {
	return yaml.Unmarshal(data, s)
}

func (s *ServiceManifest) ToYaml() ([]byte, error) {
	return yaml.Marshal(s)
}

func (s *ServiceManifest) SetServiceMeta(svc *types.Service) {

	if svc.Meta.Name == types.EmptyString {
		svc.Meta.Name = *s.Meta.Name
	}

	if s.Meta.Description != nil {
		svc.Meta.Description = *s.Meta.Description
	}

	if s.Meta.Labels != nil {
		svc.Meta.Labels = s.Meta.Labels
	}

}

func (s *ServiceManifest) SetServiceSpec(svc *types.Service) {

	tn := svc.Spec.Network.Updated
	tc := svc.Spec.Template.Updated

	defer func() {
		if s.Spec.Replicas != nil {
			svc.Status.State = types.StateProvision
			return
		}

		if tn.Before(svc.Spec.Network.Updated) || tc.Before(svc.Spec.Template.Updated) {
			svc.Status.State = types.StateProvision
			return
		}
	}()

	if s.Spec.Replicas != nil {
		svc.Spec.Replicas = *s.Spec.Replicas
	}

	if s.Spec.Network != nil {

		if s.Spec.Network.IP != nil {
			svc.Spec.Network.IP = *s.Spec.Network.IP
		}

		if len(s.Spec.Network.Ports) > 0 {

			svc.Spec.Network.Ports = make(map[uint16]string, 0)

			for _, p := range s.Spec.Network.Ports {
				mp := strings.Split(p, ":")
				var base = 10
				var size = 16
				port, err := strconv.ParseUint(mp[0], base, size)
				if err != nil {
					continue
				}
				if len(mp) == 1 {
					svc.Spec.Network.Ports[uint16(port)] = mp[0]
				}

				if len(mp) == 2 {
					svc.Spec.Network.Ports[uint16(port)] = mp[1]
				}

			}
		}

		svc.Spec.Network.Updated = time.Now()
	}

	if s.Spec.Selector != nil {

		svc.Spec.Selector.Node = s.Spec.Selector.Node

		if s.Spec.Selector.Labels != nil {
			svc.Spec.Selector.Labels = s.Spec.Selector.Labels
		}

	}

	if s.Spec.Strategy != nil {
		if s.Spec.Strategy.Type != nil {
			svc.Spec.Strategy.Type = *s.Spec.Strategy.Type
		}
	}

	if s.Spec.Template != nil {

		for _, c := range s.Spec.Template.Containers {

			var (
				f    = false
				spec *types.SpecTemplateContainer
			)

			for _, sc := range svc.Spec.Template.Containers {
				if c.Name == sc.Name {
					f = true
					spec = sc
				}
			}

			if spec == nil {
				spec = new(types.SpecTemplateContainer)
			}

			if spec.Name == types.EmptyString {
				spec.Name = c.Name
				svc.Spec.Template.Updated = time.Now()
			}

			if spec.Image.Name != c.Image.Name {
				spec.Image.Name = c.Image.Name
				svc.Spec.Template.Updated = time.Now()
			}

			if spec.Image.Secret != c.Image.Secret {
				spec.Image.Secret = c.Image.Secret
				svc.Spec.Template.Updated = time.Now()
			}

			if strings.Join(spec.Exec.Command, " ") != c.Command {
				spec.Exec.Command = strings.Split(c.Command, " ")
				svc.Spec.Template.Updated = time.Now()
			}

			if strings.Join(spec.Exec.Args, "") != strings.Join(c.Args, "") {
				spec.Exec.Args = c.Args
				svc.Spec.Template.Updated = time.Now()
			}

			if strings.Join(spec.Exec.Entrypoint, " ") != c.Entrypoint {
				spec.Exec.Entrypoint = strings.Split(c.Entrypoint, " ")
				svc.Spec.Template.Updated = time.Now()
			}

			if spec.Exec.Workdir != c.Workdir {
				spec.Exec.Workdir = c.Workdir
				svc.Spec.Template.Updated = time.Now()
			}

			for _, ce := range c.Env {
				var f = false

				for _, se := range spec.EnvVars {
					if ce.Name == se.Name {
						f = true
						if se.Value != ce.Value {
							se.Value = ce.Value
							svc.Spec.Template.Updated = time.Now()
						}

						if se.Secret.Name != ce.From.Name || se.Secret.Key != ce.From.Key {
							se.Secret.Name = ce.From.Name
							se.Secret.Key = ce.From.Key
							svc.Spec.Template.Updated = time.Now()
						}
					}
				}

				if !f {
					spec.EnvVars = append(spec.EnvVars, types.SpecTemplateContainerEnv{
						Name:  ce.Name,
						Value: ce.Value,
						Secret: types.SpecTemplateContainerEnvSecret{
							Name: ce.From.Name,
							Key:  ce.From.Key,
						},
					})
				}
			}

			var envs = make([]types.SpecTemplateContainerEnv, 0)
			for _, se := range spec.EnvVars {
				for _, ce := range c.Env {
					if ce.Name == se.Name {
						envs = append(envs, se)
						break
					}
				}
			}

			if len(spec.EnvVars) != len(envs) {
				svc.Spec.Template.Updated = time.Now()
			}
			spec.EnvVars = envs

			if c.Resources.Request.RAM != spec.Resources.Request.RAM ||
				c.Resources.Request.CPU != spec.Resources.Request.CPU {
				spec.Resources.Request.RAM = c.Resources.Request.RAM
				spec.Resources.Request.CPU = c.Resources.Request.CPU
				svc.Spec.Template.Updated = time.Now()
			}

			if c.Resources.Limits.RAM != spec.Resources.Limits.RAM ||
				c.Resources.Limits.CPU != spec.Resources.Limits.CPU {
				spec.Resources.Limits.RAM = c.Resources.Limits.RAM
				spec.Resources.Limits.CPU = c.Resources.Limits.CPU
				svc.Spec.Template.Updated = time.Now()
			}

			for _, v := range c.Volumes {

				var f = false
				for _, sv := range spec.Volumes {

					if v.Name == sv.Name {
						f = true
						if sv.Mode != v.Mode || sv.Path != v.Path {
							sv.Mode = v.Mode
							sv.Path = v.Path
							svc.Spec.Template.Updated = time.Now()
						}

					}
				}
				if !f {
					spec.Volumes = append(spec.Volumes, types.SpecTemplateContainerVolume{
						Name: v.Name,
						Mode: v.Mode,
						Path: v.Path,
					})
				}
			}

			vlms := make([]types.SpecTemplateContainerVolume, 0)
			for _, sv := range spec.Volumes {
				for _, cv := range c.Volumes {
					if sv.Name == cv.Name {
						vlms = append(vlms, sv)
						break
					}
				}
			}

			if len(vlms) != len(spec.Volumes) {
				svc.Spec.Template.Updated = time.Now()
			}

			spec.Volumes = vlms

			if !f {
				svc.Spec.Template.Containers = append(svc.Spec.Template.Containers, spec)
			}

		}

		var spcs = make([]*types.SpecTemplateContainer, 0)
		for _, ss := range svc.Spec.Template.Containers {
			for _, cs := range s.Spec.Template.Containers {
				if ss.Name == cs.Name {
					spcs = append(spcs, ss)
				}
			}
		}

		if len(spcs) != len(svc.Spec.Template.Containers) {
			svc.Spec.Template.Updated = time.Now()
		}

		svc.Spec.Template.Containers = spcs

		for _, v := range s.Spec.Template.Volumes {

			var (
				f    = false
				spec *types.SpecTemplateVolume
			)

			for _, sv := range svc.Spec.Template.Volumes {
				if v.Name == sv.Name {
					f = true
					spec = sv
				}
			}

			if spec == nil {
				spec = new(types.SpecTemplateVolume)
			}

			if spec.Name == types.EmptyString {
				spec.Name = v.Name
				svc.Spec.Template.Updated = time.Now()
			}

			if v.Type != spec.Type || v.Secret.Name != spec.Secret.Name {
				spec.Type = v.Type
				spec.Secret.Name = v.Secret.Name
				svc.Spec.Template.Updated = time.Now()
			}

			var e = true
			for _, vf := range v.Secret.Files {

				var f = false
				for _, sf := range spec.Secret.Files {
					if vf == sf {
						f = true
						break
					}
				}

				if !f {
					e = false
					break
				}

			}

			if !e {
				spec.Secret.Files = v.Secret.Files
				svc.Spec.Template.Updated = time.Now()
			}

			if !f {
				svc.Spec.Template.Volumes = append(svc.Spec.Template.Volumes, spec)
			}

		}

		var vlms = make([]*types.SpecTemplateVolume, 0)
		for _, ss := range svc.Spec.Template.Volumes {
			for _, cs := range s.Spec.Template.Volumes {
				if ss.Name == cs.Name {
					vlms = append(vlms, ss)
				}
			}
		}

		if len(vlms) != len(svc.Spec.Template.Volumes) {
			svc.Spec.Template.Updated = time.Now()
		}

		svc.Spec.Template.Volumes = vlms

	}

}

func (s *ServiceManifest) GetManifest() *types.ServiceManifest {
	sm := new(types.ServiceManifest)
	return sm
}

// swagger:ignore
// swagger:model request_service_remove
type ServiceRemoveOptions struct {
	Force bool `json:"force"`
}

// swagger:ignore
// swagger:model request_service_logs
type ServiceLogsOptions struct {
	Deployment string `json:"deployment"`
	Pod        string `json:"pod"`
	Container  string `json:"container"`
	Follow     bool   `json:"follow"`
}
