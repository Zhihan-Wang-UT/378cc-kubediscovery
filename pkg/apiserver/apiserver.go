package apiserver

import (
	"fmt"
	"strings"

	"encoding/json"

	"github.com/emicklei/go-restful"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/cloud-ark/kubediscovery/pkg/discovery"
	"net/http"
)

const GroupName = "platform-as-code"
const GroupVersion = "v1"
const KIND_QUERY_PARAM = "kind"
const INSTANCE_QUERY_PARAM = "instance"
const NAMESPACE_QUERY_PARAM = "namespace"

var (
	Scheme             = runtime.NewScheme()
	Codecs             = serializer.NewCodecFactory(Scheme)
	SchemeBuilder      = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme        = SchemeBuilder.AddToScheme
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: GroupVersion}
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion)
	return nil
}

func init() {
	utilruntime.Must(AddToScheme(Scheme))

	// Setting VersionPriority is critical in the InstallAPIGroup call (done in New())
	utilruntime.Must(Scheme.SetVersionPriority(SchemeGroupVersion))

	// TODO(devdattakulkarni) -- Following comments coming from sample-apiserver.
	// Leaving them for now.
	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: GroupVersion})

	// TODO(devdattakulkarni) -- Following comments coming from sample-apiserver.
	// Leaving them for now.
	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: GroupVersion}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)

	// Start building composition trees.
	//namespace := "" // all namespaces
	//go discovery.BuildCompositionTree(namespace)
}

type ExtraConfig struct {
	// Place you custom config here.
}

type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// DiscoveryServer contains state for a Kubernetes cluster master/api server.
type DiscoveryServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

type CompletedConfig struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

// New returns a new instance of DiscoveryServer from the given config.
func (c completedConfig) New() (*DiscoveryServer, error) {
	genericServer, err := c.GenericConfig.New("kube discovery server", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	s := &DiscoveryServer{
		GenericAPIServer: genericServer,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(GroupName, Scheme, metav1.ParameterCodec, Codecs)

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	//installKubePlusPaths(s)

	return s, nil
}

func handleResourceDetailsEndpoint(request *restful.Request, response *restful.Response) {

	resourceKind := request.QueryParameter(KIND_QUERY_PARAM)
	resourceInstance := request.QueryParameter(INSTANCE_QUERY_PARAM)
	namespace := request.QueryParameter(NAMESPACE_QUERY_PARAM)

	fmt.Printf("Kind:%s, Instance:%s\n", resourceKind, resourceInstance)
	if namespace == "" {
		namespace = "default"
	}
	resourceInfo := discovery.TotalClusterCompositions.QueryResource(resourceKind, resourceInstance, namespace)
	fmt.Printf("Resource Info:%v\n", resourceInfo)

	response.Write([]byte(resourceInfo))
}

func InstallKubePlusPaths() {//discoveryServer *DiscoveryServer) {

	fmt.Printf("Inside InstallKubePlusPaths...")
	path := "/apis/" + GroupName + "/" + GroupVersion

	ws1 := getWebService()
	ws1.Path(path).
		Consumes(restful.MIME_JSON, restful.MIME_XML).
		Produces(restful.MIME_JSON, restful.MIME_XML)

	ws1.Route(ws1.GET("/helloworld").To(handleHelloWorld))
	ws1.Route(ws1.GET("/explain").To(handleExplainEndpoint))
	ws1.Route(ws1.GET("/composition").To(handleCompositionEndpoint))
	//ws1.Route(ws1.GET("/implementation_details").To(handleImplementationDetailsEndpoint))
	//ws1.Route(ws1.GET("/usage").To(handleUsageEndpoint))
	ws1.Route(ws1.GET("/man").To(handleManPageEndpoint))
	ws1.Route(ws1.GET("/resourceDetails").To(handleResourceDetailsEndpoint))
	restful.Add(ws1)
	http.ListenAndServe(":8080", nil)
	fmt.Printf("Done installing KubePlus paths...")

	//discoveryServer.GenericAPIServer.Handler.GoRestfulContainer.Add(ws1)
}

func handleHelloWorld(request *restful.Request, response *restful.Response) {
	queryResponse := "Hello world!"
	response.Write([]byte(queryResponse))
}

func handleExplainEndpoint(request *restful.Request, response *restful.Response) {
	customResourceKind := request.QueryParameter(KIND_QUERY_PARAM)
	customResourceKind, queryKind := getQueryKind(customResourceKind)
	openAPISpec, err := discovery.GetOpenAPISpec(customResourceKind)
	queryResponse := ""
	if err != nil {
		queryResponse = "Error in retrieving OpenAPI Spec for Custom Resource:"
	} else if openAPISpec != "" {
		queryResponse = parseOpenAPISpec([]byte(openAPISpec), queryKind)
	}

	response.Write([]byte(queryResponse))
}

func handleManPageEndpoint(request *restful.Request, response *restful.Response) {
	customResourceKind := request.QueryParameter(KIND_QUERY_PARAM)

	namespace := "default"
	manPage := GetManPage(customResourceKind, namespace)

	response.Write([]byte(manPage))
}

func GetManPage(customResourceKind string, namespace string) string {
	//fmt.Printf("Custom Resource Kind:%s\n", customResourceKind)

	/*implementationDetails, err := discovery.GetImplementationDetails(customResourceKind)

	if err != nil {
		implementationDetails = "Error in retrieving implementation details for Custom Resource:"
	}

	fmt.Println("Implementation choices:%v", implementationDetails)
	*/

	manPage := discovery.GetUsageDetails(customResourceKind, namespace)
	//fmt.Println("Usage guidelines:%v", manPage)

	return manPage
}

func handleImplementationDetailsEndpoint(request *restful.Request, response *restful.Response) {
	customResourceKind := request.QueryParameter(KIND_QUERY_PARAM)
	fmt.Printf("Custom Resource Kind:%s\n", customResourceKind)

	implementationDetails, err := discovery.GetImplementationDetails(customResourceKind)

	if err != nil {
		implementationDetails = "Error in retrieving implementation details for Custom Resource:"
	}

	fmt.Println("Implementation details:%v", implementationDetails)

	response.Write([]byte(implementationDetails))
}

func handleUsageEndpoint(request *restful.Request, response *restful.Response) {
	customResourceKind := request.QueryParameter(KIND_QUERY_PARAM)
	fmt.Printf("Custom Resource Kind:%s\n", customResourceKind)

	namespace := "default"
	usageDetails := discovery.GetUsageDetails(customResourceKind, namespace)

	fmt.Println("Usage details:%v", usageDetails)

	response.Write([]byte(usageDetails))
}

func handleCompositionEndpoint(request *restful.Request, response *restful.Response) {
	resourceKind := request.QueryParameter(KIND_QUERY_PARAM)
	resourceInstance := request.QueryParameter(INSTANCE_QUERY_PARAM)
	namespace := request.QueryParameter(NAMESPACE_QUERY_PARAM)
	fmt.Printf("Kind:%s, Instance:%s\n", resourceKind, resourceInstance)
	if namespace == "" {
		namespace = "default"
	}

	discovery.BuildCompositionTree(namespace)

	compositionInfo := discovery.TotalClusterCompositions.GetCompositionsString(resourceKind,
																		  resourceInstance,
																		  namespace)
	fmt.Printf("Composition:%v\n", compositionInfo)

	response.Write([]byte(compositionInfo))
}

func getWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/apis")
	ws.Consumes("*/*")
	ws.Produces(restful.MIME_JSON, restful.MIME_XML)
	ws.ApiVersion(GroupName)
	return ws
}

func getQueryKind(input string) (string, string) {
	// Input can be Postgres.PostgresSpec.UserSpec
	// Return the last entry (i.e. UserSpec in above example)
	customResourceKind := ""
	queryKind := ""
	parts := strings.Split(input, ".")
	queryKind = parts[len(parts)-1]
	customResourceKind = parts[0]
	return customResourceKind, queryKind
}

func parseOpenAPISpec(openAPISpec []byte, customResourceKind string) string {
	var data interface{}
	retVal := ""
	err := json.Unmarshal(openAPISpec, &data)
	if err != nil {
		fmt.Printf("Error:%v\n", err)
	}

	overallMap := data.(map[string]interface{})

	definitionsMap := overallMap["definitions"].(map[string]interface{})

	queryString := "typedir." + customResourceKind
	resultMap := definitionsMap[queryString]

	result, err1 := json.Marshal(resultMap)

	if err1 != nil {
		fmt.Printf("Error:%v\n", err1)
	}

	retVal = string(result)

	return retVal
}

func getCompositions(request *restful.Request, response *restful.Response) {
	resourceName := request.PathParameter("resource-id")
	requestPath := request.Request.URL.Path
	fmt.Printf("Printing Composition\n")
	fmt.Printf("Resource Name:%s\n", resourceName)
	fmt.Printf("Request Path:%s\n", requestPath)
	//discovery.TotalClusterCompositions.PrintCompositions()
	// Path looks as follows:
	// /apis/kubediscovery.cloudark.io/v1/namespaces/default/deployments/dep1/compositions
	resourcePathSlice := strings.Split(requestPath, "/")
	resourceKind := resourcePathSlice[6] // Kind is 7th element in the slice
	resourceNamespace := resourcePathSlice[5]
	fmt.Printf("Resource Kind:%s, Resource name:%s\n", resourceKind, resourceName)

	compositionsInfo := discovery.TotalClusterCompositions.GetCompositionsString(resourceKind, resourceName, resourceNamespace)
	fmt.Printf("Compositions Info:%v", compositionsInfo)

	response.Write([]byte(compositionsInfo))
}
