package godaddy

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/b13f/terraform-provider-godaddy/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	attrRecord = "record"

	recName     = "name"
	recType     = "type"
	recData     = "data"
	recTTL      = "ttl"
	recPriority = "priority"
	recWeight   = "weight"
	recProto    = "protocol"
	recService  = "service"
	recPort     = "port"
)

type domainRecordResource struct {
	Customer string
	Domain   string
	Records  []*api.DomainRecord
}

func newDomainRecordResource(d *schema.ResourceData) (*domainRecordResource, error) {
	var err error
	r := &domainRecordResource{}

	if attr, ok := d.GetOk(attrRecord); ok {
		records := attr.(*schema.Set).List()
		r.Records = make([]*api.DomainRecord, len(records))

		for i, rec := range records {
			data := rec.(map[string]interface{})
			t := data[recType].(string)
			// if strings.EqualFold(t, api.NSType) {
			// 	nsCount++
			// }
			r.Records[i], err = api.NewDomainRecord(
				data[recName].(string),
				t,
				data[recData].(string),
				data[recTTL].(int),
				api.Priority(data[recPriority].(int)),
				api.Weight(data[recWeight].(int)),
				api.Port(data[recPort].(int)),
				api.Service(data[recService].(string)),
				api.Protocol(data[recProto].(string)))

			if err != nil {
				return r, err
			}
		}
	}

	return r, err
}

func (r *domainRecordResource) mergeRecords(list []string, factory api.RecordFactory) {
	for _, data := range list {
		record, _ := factory(data)
		r.Records = append(r.Records, record)
	}
}

func resourceDomainRecord() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceDomainRecordCreate,
		ReadContext:   resourceDomainRecordRead,
		UpdateContext: resourceDomainRecordUpdate,
		DeleteContext: resourceDomainRecordRestore,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			attrRecord: {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						recName: {
							Type:     schema.TypeString,
							Required: true,
						},
						recType: {
							Type:     schema.TypeString,
							Required: true,
						},
						recData: {
							Type:     schema.TypeString,
							Required: true,
						},
						recTTL: {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  api.DefaultTTL,
						},
						recPriority: {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  api.DefaultPriority,
						},
						recWeight: {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  api.DefaultWeight,
						},
						recService: {
							Type:     schema.TypeString,
							Optional: true,
						},
						recProto: {
							Type:     schema.TypeString,
							Optional: true,
						},
						recPort: {
							Type:     schema.TypeInt,
							Optional: true,
							Default:  api.DefaultPort,
						},
					},
				},
			},
		},
	}
}

func resourceDomainRecordRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*api.Client)
	customer := d.Get(zattrCustomer).(string)
	domain := d.Get(attrDomain).(string)
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	r, err := newDomainRecordResource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Importer support
	if domain == "" {
		r.Domain = d.Id()
		domain = r.Domain
	}

	log.Println("Fetching", domain, "records...")
	records, err := client.GetDomainRecords(customer, domain)

	if err != nil {
		return diag.FromErr(fmt.Errorf("couldn't find domain record (%s): %s", domain, err.Error()))
	}

	if err := populateResourceDataFromResponse(records, r, d); err != nil {
		return diag.FromErr(err)
	}

	if err := populateDomainInfo(client, r, d); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func resourceDomainRecordCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*api.Client)
	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics

	r, err := newDomainRecordResource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = populateDomainInfo(client, r, d); err != nil {
		return diag.FromErr(err)
	}

	log.Println("Creating", r.Domain, "domain records...")

	err = client.ReplaceDomainRecords(r.Customer, r.Domain, r.Records)

	if err != nil {
		return diag.FromErr(err)
	}
	// Implement read to populate the Terraform state to its current state after the resource creation
	resourceDomainRecordRead(ctx, d, meta)

	return diags
}

func resourceDomainRecordUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*api.Client)
	r, err := newDomainRecordResource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = populateDomainInfo(client, r, d); err != nil {
		return diag.FromErr(err)
	}

	log.Println("Updating", r.Domain, "domain records...")

	err = client.ReplaceDomainRecords(r.Customer, r.Domain, r.Records)

	if err != nil {
		return diag.FromErr(err)
	}
	// Implement read to populate the Terraform state to its current state after the resource creation
	return resourceDomainRecordRead(ctx, d, meta)
}

func resourceDomainRecordRestore(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*api.Client)
	r, err := newDomainRecordResource(d)
	if err != nil {
		return diag.FromErr(err)
	}

	if err = populateDomainInfo(client, r, d); err != nil {
		return diag.FromErr(err)
	}

	log.Println("Restoring", r.Domain, "domain records...")
	err = client.AddDomainRecords(r.Customer, r.Domain, r.Records)

	if err != nil {
		return diag.FromErr(err)
	}
	// Implement read to populate the Terraform state to its current state after the resource creation
	return resourceDomainRecordRead(ctx, d, meta)

}

func populateDomainInfo(client *api.Client, r *domainRecordResource, d *schema.ResourceData) error {
	var err error
	var domain *api.Domain

	log.Println("Fetching", r.Domain, "info...")
	domain, err = client.GetDomain(r.Customer, r.Domain)
	if err != nil {
		return fmt.Errorf("couldn't find domain (%s): %s", r.Domain, err.Error())
	}

	d.SetId(strconv.FormatInt(domain.ID, 10))

	return nil
}

func populateResourceDataFromResponse(recs []*api.DomainRecord, r *domainRecordResource, d *schema.ResourceData) error {
	aRecords := make([]string, 0)
	//nsRecords := make([]string, 0)
	domain := d.Get(attrDomain).(string)
	records := make([]*api.DomainRecord, 0)

	for _, rec := range recs {
		switch {
		// case api.IsDefaultNSRecord(rec):
		// 	nsRecords = append(nsRecords, rec.Data)
		case api.IsDefaultARecord(rec):
			aRecords = append(aRecords, rec.Data)
		default:
			records = append(records, rec)
		}
	}

	if err := d.Set(attrRecord, flattenRecords(records)); err != nil {
		return err
	}

	if domain == "" {
		d.Set(attrDomain, d.Id())
	}

	return nil
}

func flattenRecords(list []*api.DomainRecord) []map[string]interface{} {
	result := make([]map[string]interface{}, len(list))
	for i, r := range list {
		result[i] = map[string]interface{}{
			recName:     r.Name,
			recType:     r.Type,
			recData:     r.Data,
			recTTL:      r.TTL,
			recPriority: r.Priority,
			recWeight:   r.Weight,
			recPort:     r.Port,
			recService:  r.Service,
			recProto:    r.Protocol,
		}
	}
	return result
}
