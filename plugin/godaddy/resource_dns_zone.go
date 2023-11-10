package godaddy

import (
	"context"
	"fmt"
	"log"
	"strconv"
	//"strings"

	"github.com/b13f/terraform-provider-godaddy/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	zattrCustomer   = "customer"
	attrDomain      = "domain"
	attrAddresses   = "addresses"
	attrNameservers = "nameservers"
)

type domainZoneResource struct {
	Customer  string
	Domain    string
	NSRecords []string
}

func newDomainZoneResource(d *schema.ResourceData) (*domainZoneResource, error) {
	var err error
	r := &domainZoneResource{}

	if attr, ok := d.GetOk(zattrCustomer); ok {
		r.Customer = attr.(string)
	}

	if attr, ok := d.GetOk(attrDomain); ok {
		r.Domain = attr.(string)
	}

	if attr, ok := d.GetOk(attrNameservers); ok {
		for _, item := range attr.([]interface{}) {
			r.NSRecords = append(r.NSRecords, item.(string))
		}
	}

	return r, err
}

func resourceDomainZone() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDomainZoneCreate,
		ReadContext:   resourceDomainZoneRead,
		UpdateContext: resourceDomainZoneUpdate,
		DeleteContext: resourceDomainZoneRestore,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Required
			attrDomain: {
				Type:     schema.TypeString,
				Required: true,
			},
			zattrCustomer: {
				Type:     schema.TypeString,
				Optional: true,
			},
			attrNameservers: {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceDomainZoneRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*api.Client)
	//customer := d.Get(zattrCustomer).(string)
	domain := d.Get(attrDomain).(string)
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	r, err := newDomainZoneResource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("IIIIIIIIIIIIIIIIIIIIIIIIIIMMMM %+v", d.Id())

	// Importer support
	if domain == "" {
		r.Domain = d.Id()
		domain = r.Domain
	}

	log.Println("Fetching", domain, "records...")

	if err := zpopulateDomainInfo(client, r, d); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceDomainZoneCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*api.Client)
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	r, err := newDomainZoneResource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = zpopulateDomainInfo(client, r, d); err != nil {
		return diag.FromErr(err)
	}

	log.Println("Creating", r.Domain, "domain records...")

	if err != nil {
		return diag.FromErr(err)
	}
	// Implement read to populate the Terraform state to its current state after the resource creation
	resourceDomainZoneRead(ctx, d, meta)

	return diags
}

func resourceDomainZoneUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*api.Client)
	r, err := newDomainZoneResource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = zpopulateDomainInfo(client, r, d); err != nil {
		return diag.FromErr(err)
	}
	//_, _ = client.GetShoppers("")

	//err = client.UpdateDomainInfo(r.Domain, r.NSRecords)

	err = client.UpdateNSDomain(r.NSRecords, r.Customer, r.Domain)
	if err != nil {
		return diag.FromErr(err)
	}

	/*client.GetPoll(r.Domain)
	time.Sleep(10 * time.Second)
	client.GetPoll(r.Domain)
	time.Sleep(10 * time.Second)
	client.GetPoll(r.Domain)*/

	/*2023-11-06T14:49:42.620+0300 [INFO]  provider.terraform-provider-godaddy_v1.9.0: 2023/11/06 14:49:42 =====Request================
		200 OK [{"actionId":"3935e84f-6e88-4985-a575-a08f50f9f88a","createdAt":"2023-11-06T11:49:41.629Z","modifiedAt":"2023-11-06T11:49:41.869Z","origination":"USER","requestId":"7skftYDDAgDKDcNarCD1gB","startedAt":"2023-11-06T11:49:41.863Z","status":"ACCEPTED","type":"DOMAIN_UPDATE_NAME_SERVERS"}]: timestamp=2023-11-06T14:49:42.620+0300
	godaddy_domain_zone.api: Still modifying... [id=403906868, 10s elapsed]
	2023-11-06T14:49:52.623+0300 [INFO]  provider.terraform-provider-godaddy_v1.9.0: 2023/11/06 14:49:52 PPPPOOOOOOLLLLLLOOLLLLLL: timestamp=2023-11-06T14:49:52.623+0300
	2023-11-06T14:49:53.475+0300 [INFO]  provider.terraform-provider-godaddy_v1.9.0: 2023/11/06 14:49:53 =====Request================
	200 OK [{"actionId":"3935e84f-6e88-4985-a575-a08f50f9f88a","completedAt":"2023-11-06T11:49:43.970Z","createdAt":"2023-11-06T11:49:41.629Z","modifiedAt":"2023-11-06T11:49:44.287Z","origination":"USER","reason":{"code":"INVALID_BODY","message":"Nameserver change is not allowed for the domain"},"requestId":"7skftYDDAgDKDcNarCD1gB","startedAt":"2023-11-06T11:49:41.863Z","status":"FAILED","type":"DOMAIN_UPDATE_NAME_SERVERS"}]: timestamp=2023-11-06T14:49:53.475+0300
	godaddy_domain_zone.api: Still modifying... [id=403906868, 20s elapsed]*/

	// Implement read to populate the Terraform state to its current state after the resource creation
	return resourceDomainZoneRead(ctx, d, meta)
}

func resourceDomainZoneRestore(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*api.Client)
	r, err := newDomainZoneResource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = zpopulateDomainInfo(client, r, d); err != nil {
		return diag.FromErr(err)
	}

	if err != nil {
		return diag.FromErr(err)
	}
	// Implement read to populate the Terraform state to its current state after the resource creation
	return resourceDomainZoneRead(ctx, d, meta)

}

func zpopulateDomainInfo(client *api.Client, r *domainZoneResource, d *schema.ResourceData) error {
	var err error
	var domain *api.Domain

	log.Println("Fetching", r.Domain, "info...")
	domain, err = client.GetDomain(r.Customer, r.Domain)
	if err != nil {
		return fmt.Errorf("couldn't find domain (%s): %s", r.Domain, err.Error())
	}

	d.SetId(strconv.FormatInt(domain.ID, 10))

	d.Set(attrNameservers, domain.NameServers)

	d.Set(attrDomain, r.Domain)

	return nil
}
