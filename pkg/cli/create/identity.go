package create

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/scheme"
	"k8s.io/kubectl/pkg/util"
	"k8s.io/kubectl/pkg/util/templates"

	userv1 "github.com/openshift/api/user/v1"
	userv1client "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
)

var (
	identityLong = templates.LongDesc(`
		This command can be used to create an identity object.

		Typically, identities are created automatically during login. If automatic
		creation is disabled (by using the "lookup" mapping method), identities must
		be created manually.

		Corresponding user and user identity mapping objects must also be created
		to allow logging in with the created identity.
	`)

	identityExample = templates.Examples(`
		# Create an identity with identity provider "acme_ldap" and the identity provider username "adamjones"
		oc create identity acme_ldap:adamjones
	`)
)

type CreateIdentityOptions struct {
	CreateSubcommandOptions *CreateSubcommandOptions

	ProviderName     string
	ProviderUserName string

	IdentityClient userv1client.IdentitiesGetter
}

// NewCmdCreateIdentity is a macro command to create a new identity
func NewCmdCreateIdentity(f genericclioptions.RESTClientGetter, streams genericiooptions.IOStreams) *cobra.Command {
	o := &CreateIdentityOptions{
		CreateSubcommandOptions: NewCreateSubcommandOptions(streams),
	}
	cmd := &cobra.Command{
		Use:     "identity <PROVIDER_NAME>:<PROVIDER_USER_NAME>",
		Short:   "Manually create an identity (only needed if automatic creation is disabled)",
		Long:    identityLong,
		Example: identityExample,
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(cmd, f, args))
			cmdutil.CheckErr(o.Run())
		},
	}

	o.CreateSubcommandOptions.AddFlags(cmd)
	cmdutil.AddDryRunFlag(cmd)

	return cmd
}

func (o *CreateIdentityOptions) Complete(cmd *cobra.Command, f genericclioptions.RESTClientGetter, args []string) error {
	clientConfig, err := f.ToRESTConfig()
	if err != nil {
		return err
	}
	o.IdentityClient, err = userv1client.NewForConfig(clientConfig)
	if err != nil {
		return err
	}

	if err := o.CreateSubcommandOptions.Complete(f, cmd, args); err != nil {
		return err
	}

	parts := strings.Split(o.CreateSubcommandOptions.Name, ":")
	if len(parts) != 2 {
		return fmt.Errorf("identity name in the format <PROVIDER_NAME>:<PROVIDER_USER_NAME> is required")
	}
	o.ProviderName = parts[0]
	o.ProviderUserName = parts[1]

	return nil
}

func (o *CreateIdentityOptions) Run() error {
	identity := &userv1.Identity{
		// this is ok because we know exactly how we want to be serialized
		TypeMeta:         metav1.TypeMeta{APIVersion: userv1.SchemeGroupVersion.String(), Kind: "Identity"},
		ProviderName:     o.ProviderName,
		ProviderUserName: o.ProviderUserName,
	}

	if err := util.CreateOrUpdateAnnotation(o.CreateSubcommandOptions.CreateAnnotation, identity, scheme.DefaultJSONEncoder()); err != nil {
		return err
	}

	if o.CreateSubcommandOptions.DryRunStrategy != cmdutil.DryRunClient {
		var err error
		identity, err = o.IdentityClient.Identities().Create(context.TODO(), identity, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}

	return o.CreateSubcommandOptions.Printer.PrintObj(identity, o.CreateSubcommandOptions.Out)
}
