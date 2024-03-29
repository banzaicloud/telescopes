// Code generated by go-swagger; DO NOT EDIT.

package recommend

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/swag"

	strfmt "github.com/go-openapi/strfmt"
)

// NewRecommendMultiClusterParams creates a new RecommendMultiClusterParams object
// with the default values initialized.
func NewRecommendMultiClusterParams() *RecommendMultiClusterParams {
	var ()
	return &RecommendMultiClusterParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewRecommendMultiClusterParamsWithTimeout creates a new RecommendMultiClusterParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewRecommendMultiClusterParamsWithTimeout(timeout time.Duration) *RecommendMultiClusterParams {
	var ()
	return &RecommendMultiClusterParams{

		timeout: timeout,
	}
}

// NewRecommendMultiClusterParamsWithContext creates a new RecommendMultiClusterParams object
// with the default values initialized, and the ability to set a context for a request
func NewRecommendMultiClusterParamsWithContext(ctx context.Context) *RecommendMultiClusterParams {
	var ()
	return &RecommendMultiClusterParams{

		Context: ctx,
	}
}

// NewRecommendMultiClusterParamsWithHTTPClient creates a new RecommendMultiClusterParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewRecommendMultiClusterParamsWithHTTPClient(client *http.Client) *RecommendMultiClusterParams {
	var ()
	return &RecommendMultiClusterParams{
		HTTPClient: client,
	}
}

/*
RecommendMultiClusterParams contains all the parameters to send to the API endpoint
for the recommend multi cluster operation typically these are written to a http.Request
*/
type RecommendMultiClusterParams struct {

	/*Provider*/
	Provider *string
	/*Services*/
	Services []string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the recommend multi cluster params
func (o *RecommendMultiClusterParams) WithTimeout(timeout time.Duration) *RecommendMultiClusterParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the recommend multi cluster params
func (o *RecommendMultiClusterParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the recommend multi cluster params
func (o *RecommendMultiClusterParams) WithContext(ctx context.Context) *RecommendMultiClusterParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the recommend multi cluster params
func (o *RecommendMultiClusterParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the recommend multi cluster params
func (o *RecommendMultiClusterParams) WithHTTPClient(client *http.Client) *RecommendMultiClusterParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the recommend multi cluster params
func (o *RecommendMultiClusterParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithProvider adds the provider to the recommend multi cluster params
func (o *RecommendMultiClusterParams) WithProvider(provider *string) *RecommendMultiClusterParams {
	o.SetProvider(provider)
	return o
}

// SetProvider adds the provider to the recommend multi cluster params
func (o *RecommendMultiClusterParams) SetProvider(provider *string) {
	o.Provider = provider
}

// WithServices adds the services to the recommend multi cluster params
func (o *RecommendMultiClusterParams) WithServices(services []string) *RecommendMultiClusterParams {
	o.SetServices(services)
	return o
}

// SetServices adds the services to the recommend multi cluster params
func (o *RecommendMultiClusterParams) SetServices(services []string) {
	o.Services = services
}

// WriteToRequest writes these params to a swagger request
func (o *RecommendMultiClusterParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Provider != nil {

		// query param provider
		var qrProvider string
		if o.Provider != nil {
			qrProvider = *o.Provider
		}
		qProvider := qrProvider
		if qProvider != "" {
			if err := r.SetQueryParam("provider", qProvider); err != nil {
				return err
			}
		}

	}

	valuesServices := o.Services

	joinedServices := swag.JoinByFormat(valuesServices, "")
	// query array param services
	if err := r.SetQueryParam("services", joinedServices...); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
