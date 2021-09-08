package generator

import (
	"fmt"

	fluxcd_io_image_reflector_v1beta1 "github.com/nais/liberator/pkg/apis/fluxcd.io/image-reflector/v1beta1"
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateImagePolicy(application skatteetaten_no_v1alpha1.Application) (*fluxcd_io_image_reflector_v1beta1.ImagePolicy, error) {

	imagePolicyConfiguration := application.Spec.ImagePolicy
	if imagePolicyConfiguration == nil || imagePolicyConfiguration.Enabled == false {
		return nil, nil
	}

	hasBranch := imagePolicyConfiguration.Branch != ""
	hasVersion := imagePolicyConfiguration.Semver != ""

	if hasBranch && hasVersion {
		return nil, fmt.Errorf("specify either version or branch, not both")
	}

	if !hasBranch && !hasVersion && imagePolicyConfiguration.Enabled == true {
		return nil, fmt.Errorf("invalid specification, specify either branchName or semVer range or disable imagePolicy")
	}

	var tags *fluxcd_io_image_reflector_v1beta1.TagFilter
	var choice fluxcd_io_image_reflector_v1beta1.ImagePolicyChoice

	if imagePolicyConfiguration.Branch != "" {
		choice = fluxcd_io_image_reflector_v1beta1.ImagePolicyChoice{
			Numerical: &fluxcd_io_image_reflector_v1beta1.NumericalPolicy{
				Order: "asc",
			},
		}
		//TODO: validate branch name?
		tags = &fluxcd_io_image_reflector_v1beta1.TagFilter{
			Extract: "$date$time$number",
			Pattern: fmt.Sprintf(`^SNAPSHOT-%s-(?P<date>[0-9]+)\.(?P<time>[0-9]+)-(?P<number>[0-9]+)`, imagePolicyConfiguration.Branch),
		}
	} else if imagePolicyConfiguration.Semver != "" {
		choice = fluxcd_io_image_reflector_v1beta1.ImagePolicyChoice{
			//TODO: validate semver range?
			SemVer: &fluxcd_io_image_reflector_v1beta1.SemVerPolicy{
				Range: imagePolicyConfiguration.Semver,
			},
		}
	}

	imagePolicyName := fmt.Sprintf("%s-%s", application.Name, application.Namespace)

	imagePolicy := &fluxcd_io_image_reflector_v1beta1.ImagePolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "image.toolkit.fluxcd.io/v1beta1",
			Kind:       "ImagePolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      imagePolicyName,
			Namespace: "flux-system",
			Labels:    application.StandardLabels(),
		},
		Spec: fluxcd_io_image_reflector_v1beta1.ImagePolicySpec{
			ImageRepositoryRef: fluxcd_io_image_reflector_v1beta1.LocalObjectReference{Name: application.Name},
			Policy:             choice,
			FilterTags:         tags,
		},
	}
	return imagePolicy, nil
}
