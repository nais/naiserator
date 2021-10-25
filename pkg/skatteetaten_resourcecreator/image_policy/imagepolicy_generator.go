package image_policy

import (
	"fmt"

	fluxcd_io_image_reflector_v1beta1 "github.com/nais/liberator/pkg/apis/fluxcd.io/image-reflector/v1beta1"
	skatteetaten_no_v1alpha1 "github.com/nais/liberator/pkg/apis/nebula.skatteetaten.no/v1alpha1"
	"github.com/nais/naiserator/pkg/resourcecreator/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Source interface {
	resource.Source
	GetImagePolicy() *skatteetaten_no_v1alpha1.ImagePolicyConfig
}

func Create(app Source, ast *resource.Ast) error {
	imagePolicy := app.GetImagePolicy()

	if imagePolicy == nil  {
		return nil
	}

	hasBranch := imagePolicy.Branch != ""
	hasVersion := imagePolicy.Semver != ""

	if hasBranch && hasVersion {
		return fmt.Errorf("specify either version or branch, not both")
	}

	if !hasBranch && !hasVersion  {
		return fmt.Errorf("invalid specification, specify either branchName or semVer range or disable imagePolicy")
	}

	var tags *fluxcd_io_image_reflector_v1beta1.TagFilter
	var choice fluxcd_io_image_reflector_v1beta1.ImagePolicyChoice

	if imagePolicy.Branch != "" {
		choice = fluxcd_io_image_reflector_v1beta1.ImagePolicyChoice{
			Numerical: &fluxcd_io_image_reflector_v1beta1.NumericalPolicy{
				Order: "asc",
			},
		}
		//TODO: validate branch name?
		tags = &fluxcd_io_image_reflector_v1beta1.TagFilter{
			Extract: "$date$time$number",
			Pattern: fmt.Sprintf(`^SNAPSHOT-%s-(?P<date>[0-9]+)\.(?P<time>[0-9]+)-(?P<number>[0-9]+)`, imagePolicy.Branch),
		}
	} else if imagePolicy.Semver != "" {
		choice = fluxcd_io_image_reflector_v1beta1.ImagePolicyChoice{
			//TODO: validate semver range?
			SemVer: &fluxcd_io_image_reflector_v1beta1.SemVerPolicy{
				Range: imagePolicy.Semver,
			},
		}
	}

	imagePolicyName := fmt.Sprintf("%s-%s", app.GetName(), app.GetNamespace())

	policy := &fluxcd_io_image_reflector_v1beta1.ImagePolicy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "image.toolkit.fluxcd.io/v1beta1",
			Kind:       "ImagePolicy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      imagePolicyName,
			Namespace: "flux-system",
		},
		Spec: fluxcd_io_image_reflector_v1beta1.ImagePolicySpec{
			ImageRepositoryRef: fluxcd_io_image_reflector_v1beta1.LocalObjectReference{Name: app.GetName()},
			Policy:             choice,
			FilterTags:         tags,
		},
	}
	ast.AppendOperation(resource.OperationCreateOrUpdate, policy)

	return nil
}
