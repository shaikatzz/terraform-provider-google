// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0
package kms_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-provider-google/google/acctest"
)

func TestAccDataSourceGoogleKmsCryptoKeys_basic(t *testing.T) {
	kms := acctest.BootstrapKMSKey(t)

	acctest.VcrTest(t, resource.TestCase{
		PreCheck:                 func() { acctest.AccTestPreCheck(t) },
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceGoogleKmsCryptoKeys_basic(kms.KeyRing.Name),
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.google_kms_crypto_keys.all_keys_in_ring", "id", regexp.MustCompile(kms.KeyRing.Name)),
					resource.TestCheckResourceAttr("data.google_kms_crypto_keys.all_keys_in_ring", "key_ring", kms.KeyRing.Name),
					resource.TestMatchResourceAttr("data.google_kms_crypto_keys.all_keys_in_ring", "keys.#", regexp.MustCompile("[1-9]+[0-9]*")),
				),
			},
		},
	})
}

func testAccDataSourceGoogleKmsCryptoKeys_basic(keyRingName string) string {
	return fmt.Sprintf(`
data "google_kms_crypto_keys" "all_keys_in_ring" {
  key_ring = "%s"
}
`, keyRingName)
}
