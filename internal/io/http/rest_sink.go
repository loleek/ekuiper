// Copyright 2022-2023 EMQ Technologies Co., Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"fmt"
	"github.com/lf-edge/ekuiper/internal/conf"
	"github.com/lf-edge/ekuiper/internal/pkg/httpx"
	"github.com/lf-edge/ekuiper/pkg/api"
	"github.com/lf-edge/ekuiper/pkg/errorx"
	"net/http"
	"strings"
)

type RestSink struct {
	ClientConf
}

func (ms *RestSink) Configure(ps map[string]interface{}) error {
	conf.Log.Infof("Initialized rest sink with configurations %#v.", ps)
	return ms.InitConf("", ps)
}

func (ms *RestSink) Open(ctx api.StreamContext) error {
	ctx.GetLogger().Infof("Opening HTTP pull source with conf %+v", ms.config)
	return nil
}

type MultiErrors []error

func (me MultiErrors) AddError(err error) MultiErrors {
	me = append(me, err)
	return me
}

func (me MultiErrors) Error() string {
	s := make([]string, len(me))
	for i, v := range me {
		s = append(s, fmt.Sprintf("Error %d with info %s. \n", i, v))
	}
	return strings.Join(s, "  ")
}

func (ms *RestSink) Collect(ctx api.StreamContext, item interface{}) error {
	logger := ctx.GetLogger()
	logger.Debugf("rest sink receive %s", item)
	output, transed, err := ctx.TransformOutput(item)
	if err != nil {
		logger.Warnf("rest sink decode data error: %v", err)
		return nil
	}
	var d = item
	if transed {
		d = output
	}
	resp, err := ms.Send(ctx, d, logger)
	if err != nil {
		return fmt.Errorf("rest sink fails to send out the data: %s", err)
	} else {
		logger.Debugf("rest sink got response %v", resp)
		_, b, err := ms.parseResponse(ctx, resp, ms.config.DebugResp, nil)
		if err != nil {
			return fmt.Errorf("%s: http error %s", errorx.IOErr, string(b))
		}
		if ms.config.DebugResp {
			logger.Infof("Response raw content: %s\n", string(b))
		}
	}
	return nil
}

func (ms *RestSink) Send(ctx api.StreamContext, v interface{}, logger api.Logger) (*http.Response, error) {
	bodyType, err := ctx.ParseTemplate(ms.config.BodyType, v)
	if err != nil {
		return nil, err
	}
	method, err := ctx.ParseTemplate(ms.config.Method, v)
	if err != nil {
		return nil, err
	}
	u, err := ctx.ParseTemplate(ms.config.Url, v)
	if err != nil {
		return nil, err
	}
	headers, err := ms.parseHeaders(ctx, v)
	if err != nil {
		return nil, fmt.Errorf("rest sink headers template decode error: %v", err)
	}
	return httpx.Send(logger, ms.client, bodyType, method, u, headers, ms.config.SendSingle, v)
}

func (ms *RestSink) Close(ctx api.StreamContext) error {
	logger := ctx.GetLogger()
	logger.Infof("Closing rest sink")
	return nil
}