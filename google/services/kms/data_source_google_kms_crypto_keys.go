// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
package kms

import (
	"fmt"
	"log"
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-google/google/tpgresource"
	transport_tpg "github.com/hashicorp/terraform-provider-google/google/transport"
)

func DataSourceGoogleKmsCryptoKeys() *schema.Resource {
	dsSchema := tpgresource.DatasourceSchemaFromResourceSchema(ResourceKMSCryptoKey().Schema)
	tpgresource.AddOptionalFieldsToSchema(dsSchema, "name")
	tpgresource.AddOptionalFieldsToSchema(dsSchema, "key_ring")

	return &schema.Resource{
		Read: dataSourceGoogleKmsCryptoKeysRead,
		Schema: map[string]*schema.Schema{
			"key_ring": {
				Type:     schema.TypeString,
				Required: true,
				Description: `The KeyRing that the keys belongs to.
Format: ''projects/{{project}}/locations/{{location}}/keyRings/{{keyRing}}''.`,
			},
			"filter": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "",
			},
			"keys": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "",
				Elem: &schema.Resource{
					Schema: dsSchema,
				},
			},
		},
	}
}

func dataSourceGoogleKmsCryptoKeysRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*transport_tpg.Config)

	keyRingId, err := parseKmsKeyRingId(d.Get("key_ring").(string), config)
	if err != nil {
		return err
	}

	id := fmt.Sprintf("%s/cryptoKeys", keyRingId.KeyRingId())
	d.SetId(id)

	res, err := dataSourceKMSCryptoKeysList(d, meta)
	if err != nil {
		return err
	}

	keys := res["cryptoKeys"].([]interface{})

	if err := d.Set("keys", flattenKMSKeysList(d, config, keys)); err != nil {
		return fmt.Errorf("error setting keys: %s", err)
	}

	if err := tpgresource.SetDataSourceLabels(d); err != nil {
		return err
	}

	if d.Id() == "" {
		return fmt.Errorf("%s not found", id)
	}
	return nil
}

func dataSourceKMSCryptoKeysList(d *schema.ResourceData, meta interface{}) (map[string]interface{}, error) {
	config := meta.(*transport_tpg.Config)
	userAgent, err := tpgresource.GenerateUserAgentString(d, config.UserAgent)
	if err != nil {
		return nil, err
	}

	url, err := tpgresource.ReplaceVars(d, config, "{{KMSBasePath}}{{key_ring}}/cryptoKeys")
	if err != nil {
		return nil, err
	}

	billingProject := ""

	if parts := regexp.MustCompile(`projects\/([^\/]+)\/`).FindStringSubmatch(url); parts != nil {
		billingProject = parts[1]
	}

	// err == nil indicates that the billing_project value was found
	if bp, err := tpgresource.GetBillingProject(d, config); err == nil {
		billingProject = bp
	}

	headers := make(http.Header)
	res, err := transport_tpg.SendRequest(transport_tpg.SendRequestOptions{
		Config:    config,
		Method:    "GET",
		Project:   billingProject,
		RawURL:    url,
		UserAgent: userAgent,
		Headers:   headers,
	})
	if err != nil {
		return nil, transport_tpg.HandleNotFoundError(err, d, fmt.Sprintf("KMSCryptoKeys %q", d.Id()))
	}

	if res == nil {
		// Decoding the object has resulted in it being gone. It may be marked deleted
		log.Printf("[DEBUG] Removing KMSCryptoKey because it no longer exists.")
		d.SetId("")
		return nil, nil
	}
	return res, nil
}

// flattenKMSKeysList flattens a list of crypto keys from a given crypto key ring
func flattenKMSKeysList(d *schema.ResourceData, config *transport_tpg.Config, keysList []interface{}) []interface{} {
	var keys []interface{}
	for _, k := range keysList {
		key := k.(map[string]interface{})

		data := map[string]interface{}{}
		data["name"] = key["name"]
		data["labels"] = flattenKMSCryptoKeyLabels(key["labels"], d, config)
		data["primary"] = flattenKMSCryptoKeyPrimary(key["primary"], d, config)
		data["purpose"] = flattenKMSCryptoKeyPurpose(key["purpose"], d, config)
		data["rotation_period"] = flattenKMSCryptoKeyRotationPeriod(key["rotationPeriod"], d, config)
		data["version_template"] = flattenKMSCryptoKeyVersionTemplate(key["versionTemplate"], d, config)
		data["destroy_scheduled_duration"] = flattenKMSCryptoKeyDestroyScheduledDuration(key["destroyScheduledDuration"], d, config)
		data["import_only"] = flattenKMSCryptoKeyImportOnly(key["importOnly"], d, config)
		data["crypto_key_backend"] = flattenKMSCryptoKeyCryptoKeyBackend(key["cryptoKeyBackend"], d, config)
		keys = append(keys, data)
	}

	return keys
}

// func flattenKMSCryptoKeyName(v interface{}) interface{} {
// 	if v == nil {
// 		return v
// 	}
// }
