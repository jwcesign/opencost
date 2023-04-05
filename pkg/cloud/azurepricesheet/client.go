package azurepricesheet

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armruntime "github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
)

const (
	moduleName    = "armconsumption"
	moduleVersion = "v1.0.0"
)

// At the moment the consumption pricesheet download API is not a)
// documented or b) supported by the SDK. This is an implementation of
// a client in the style of the Azure go SDK - once the API is
// supported this will be removed.

// PriceSheetClient contains the methods for the PriceSheet group.
// Don't use this type directly, use NewPriceSheetClient() instead.
type PriceSheetClient struct {
	host             string
	billingAccountID string
	pl               runtime.Pipeline
}

// NewClient creates a new instance of PriceSheetClient with the specified values.
// subscriptionID - Azure Subscription ID.
// credential - used to authorize requests. Usually a credential from azidentity.
// options - pass nil to accept the default values.
func NewClient(billingAccountID string, credential azcore.TokenCredential, options *arm.ClientOptions) (*PriceSheetClient, error) {
	if options == nil {
		options = &arm.ClientOptions{}
	}
	ep := cloud.AzurePublic.Services[cloud.ResourceManager].Endpoint
	if c, ok := options.Cloud.Services[cloud.ResourceManager]; ok {
		ep = c.Endpoint
	}
	pl, err := armruntime.NewPipeline(moduleName, moduleVersion, credential, runtime.PipelineOptions{}, options)
	if err != nil {
		return nil, err
	}
	client := &PriceSheetClient{
		billingAccountID: billingAccountID,
		host:             ep,
		pl:               pl,
	}
	return client, nil
}

func (client *PriceSheetClient) BeginDownloadByBillingPeriod(ctx context.Context, billingPeriodName string) (*runtime.Poller[PriceSheetClientDownloadResponse], error) {
	resp, err := client.downloadByBillingPeriodOperation(ctx, billingPeriodName)
	if err != nil {
		return nil, err
	}
	return runtime.NewPoller[PriceSheetClientDownloadResponse](resp, client.pl, nil)
}

type PriceSheetClientDownloadResponse struct {
	ID         string                             `json:"id"`
	Name       string                             `json:"name"`
	StartTime  time.Time                          `json:"startTime"`
	EndTime    time.Time                          `json:"endTime"`
	Status     string                             `json:"status"`
	Properties PriceSheetClientDownloadProperties `json:"properties"`
}

type PriceSheetClientDownloadProperties struct {
	DownloadURL string `json:"downloadUrl"`
	ValidTill   string `json:"validTill"`
}

// GetByBillingPeriod - Get the price sheet for a scope by subscriptionId and billing period. Price sheet is available via
// this API only for May 1, 2014 or later.
// If the operation fails it returns an *azcore.ResponseError type.
// Generated from API version 2021-10-01
// billingPeriodName - Billing Period Name.
// options - PriceSheetClientGetByBillingPeriodOptions contains the optional parameters for the PriceSheetClient.GetByBillingPeriod
// method.
func (client *PriceSheetClient) downloadByBillingPeriodOperation(ctx context.Context, billingPeriodName string) (*http.Response, error) {
	req, err := client.downloadByBillingPeriodCreateRequest(ctx, billingPeriodName)
	if err != nil {
		return nil, err
	}
	resp, err := client.pl.Do(req)
	if err != nil {
		return nil, err
	}
	if !runtime.HasStatusCode(resp, http.StatusOK, http.StatusAccepted) {
		return nil, runtime.NewResponseError(resp)
	}
	return resp, nil
}

// getByBillingPeriodCreateRequest creates the GetByBillingPeriod request.
func (client *PriceSheetClient) downloadByBillingPeriodCreateRequest(ctx context.Context, billingPeriodName string) (*policy.Request, error) {
	urlPath := "/providers/Microsoft.Billing/billingAccounts/{billingAccountID}/billingPeriods/{billingPeriodName}/providers/Microsoft.Consumption/pricesheets/download"
	if client.billingAccountID == "" {
		return nil, errors.New("parameter client.billingAccountID cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{billingAccountID}", url.PathEscape(client.billingAccountID))
	if billingPeriodName == "" {
		return nil, errors.New("parameter billingPeriodName cannot be empty")
	}
	urlPath = strings.ReplaceAll(urlPath, "{billingPeriodName}", url.PathEscape(billingPeriodName))
	req, err := runtime.NewRequest(ctx, http.MethodGet, runtime.JoinPaths(client.host, urlPath))
	if err != nil {
		return nil, err
	}
	reqQP := req.Raw().URL.Query()
	reqQP.Set("api-version", "2022-06-01")
	reqQP.Set("ln", "en")
	req.Raw().URL.RawQuery = reqQP.Encode()
	req.Raw().Header["Accept"] = []string{"*/*"}
	return req, nil
}
