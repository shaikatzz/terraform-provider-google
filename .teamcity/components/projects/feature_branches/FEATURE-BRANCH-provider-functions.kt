/*
 * Copyright (c) HashiCorp, Inc.
 * SPDX-License-Identifier: MPL-2.0
 */

// This file is controlled by MMv1, any changes made here will be overwritten

package feature_branches

import ProviderNameBeta
import ProviderNameGa
import builds.*
import generated.PackagesListBeta
import generated.PackagesListGa
import jetbrains.buildServer.configs.kotlin.Project
import jetbrains.buildServer.configs.kotlin.vcs.GitVcsRoot
import replaceCharsId

const val featureBranchProviderFunctionsName = "FEATURE-BRANCH-provider-functions"

// VCS Roots specifically for pulling code from the feature branches in the downstream and upstream repos
object HashicorpVCSRootGa_featureBranchProviderFunctions: GitVcsRoot({
    name = "VCS root for the hashicorp/terraform-provider-${ProviderNameGa} repo @ refs/heads/${featureBranchProviderFunctionsName}"
    url = "https://github.com/hashicorp/terraform-provider-${ProviderNameGa}"
    branch = "refs/heads/${featureBranchProviderFunctionsName}"
    branchSpec = "" // empty as we'll access no other branches
})

object HashicorpVCSRootBeta_featureBranchProviderFunctions: GitVcsRoot({
    name = "VCS root for the hashicorp/terraform-provider-${ProviderNameBeta} repo @ refs/heads/${featureBranchProviderFunctionsName}"
    url = "https://github.com/hashicorp/terraform-provider-${ProviderNameBeta}"
    branch = "refs/heads/${featureBranchProviderFunctionsName}"
    branchSpec = "" // empty as we'll access no other branches
})

object ModularMagicianVCSRootGa_featureBranchProviderFunctions: GitVcsRoot({
    name = "VCS root for the modular-magician/terraform-provider-${ProviderNameGa} repo @ refs/heads/${featureBranchProviderFunctionsName}"
    url = "https://github.com/modular-magician/terraform-provider-${ProviderNameGa}"
    branch = "refs/heads/${featureBranchProviderFunctionsName}"
    branchSpec = "" // empty as we'll access no other branches
})

object ModularMagicianVCSRootBeta_featureBranchProviderFunctions: GitVcsRoot({
    name = "VCS root for the modular-magician/terraform-provider-${ProviderNameBeta} repo @ refs/heads/${featureBranchProviderFunctionsName}"
    url = "https://github.com/modular-magician/terraform-provider-${ProviderNameBeta}"
    branch = "refs/heads/${featureBranchProviderFunctionsName}"
    branchSpec = "" // empty as we'll access no other branches
})

fun featureBranchProviderFunctionSubProject(allConfig: AllContextParameters): Project {

    val projectId = replaceCharsId(featureBranchProviderFunctionsName)

    val sharedResourcesEmpty: List<String> = listOf() // No locking when testing functions

    var parentId: String = "" // To be overwritten when each build config is generated below.
    var packageName = "functions"

    // GA
    val gaConfig = getGaAcceptanceTestConfig(allConfig)

    // How to make only build configuration to the relevant package(s)
    val functionPackageGa = PackagesListGa.getValue(packageName)

    // Enable testing using hashicorp/terraform-provider-google
    parentId = "${projectId}_HC_GA"
    val buildConfigHashiCorpGa = BuildConfigurationForSinglePackage(packageName, functionPackageGa.getValue("path"), "Provider-Defined Functions (GA provider, HashiCorp downstream)", ProviderNameGa, parentId, HashicorpVCSRootGa_featureBranchProviderFunctions, sharedResourcesEmpty, gaConfig)
    // Enable testing using modular-magician/terraform-provider-google
    parentId = "${projectId}_MM_GA"
    val buildConfigModularMagicianGa = BuildConfigurationForSinglePackage(packageName, functionPackageGa.getValue("path"), "Provider-Defined Functions (GA provider, MM upstream)", ProviderNameGa, parentId, ModularMagicianVCSRootGa_featureBranchProviderFunctions, sharedResourcesEmpty, gaConfig)


    // Beta
    val betaConfig = getBetaAcceptanceTestConfig(allConfig)
    val functionPackageBeta = PackagesListBeta.getValue("functions")

    // Enable testing using hashicorp/terraform-provider-google-beta
    parentId = "${projectId}_HC_BETA"
    val buildConfigHashiCorpBeta = BuildConfigurationForSinglePackage(packageName, functionPackageBeta.getValue("path"), "Provider-Defined Functions (Beta provider, HashiCorp downstream)", ProviderNameBeta, parentId, HashicorpVCSRootBeta_featureBranchProviderFunctions, sharedResourcesEmpty, betaConfig)

    // Enable testing using modular-magician/terraform-provider-google-beta
    parentId = "${projectId}_MM_BETA"
    val buildConfigModularMagicianBeta = BuildConfigurationForSinglePackage(packageName, functionPackageBeta.getValue("path"), "Provider-Defined Functions (Beta provider, MM upstream)", ProviderNameBeta, parentId, ModularMagicianVCSRootBeta_featureBranchProviderFunctions, sharedResourcesEmpty, betaConfig)

    return Project{
        id(projectId)
        name = featureBranchProviderFunctionsName
        description = "Subproject for testing feature branch ${featureBranchProviderFunctionsName}"

        // Register all feature branch-specific VCS roots in the project
        vcsRoot(HashicorpVCSRootGa_featureBranchProviderFunctions)
        vcsRoot(HashicorpVCSRootBeta_featureBranchProviderFunctions)
        vcsRoot(ModularMagicianVCSRootGa_featureBranchProviderFunctions)
        vcsRoot(ModularMagicianVCSRootBeta_featureBranchProviderFunctions)

        // Register all build configs in the project
        buildType(buildConfigHashiCorpGa)
        buildType(buildConfigModularMagicianGa)
        buildType(buildConfigHashiCorpBeta)
        buildType(buildConfigModularMagicianBeta)

        params {
            readOnlySettings()
        }
    }
}