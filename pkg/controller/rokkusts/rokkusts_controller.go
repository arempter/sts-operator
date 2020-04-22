package rokkusts

import (
	"context"
	"fmt"
	rokkustsv1alpha1 "github.com/arempter/sts-operator/pkg/apis/rokkusts/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/auditregistration/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
)

var log = logf.Log.WithName("controller_rokkusts")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new RokkuSts Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRokkuSts{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rokkusts-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RokkuSts
	err = c.Watch(&source.Kind{Type: &rokkustsv1alpha1.RokkuSts{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner RokkuSts
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &rokkustsv1alpha1.RokkuSts{},
	})
	if err != nil {
		return err
	}
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &rokkustsv1alpha1.RokkuSts{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRokkuSts implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRokkuSts{}

// ReconcileRokkuSts reconciles a RokkuSts object
type ReconcileRokkuSts struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a RokkuSts object and makes changes based on the state read
// and what is in the RokkuSts.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRokkuSts) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	//reqLogger.Info("Reconciling RokkuSts")

	// Fetch the RokkuSts instance
	instance := &rokkustsv1alpha1.RokkuSts{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not stsFound, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// sts deployment & service
	r.reconcileDeploymentForSts(context.TODO(), instance, reqLogger)
	r.reconcileServiceFor(context.TODO(), instance, "rokkusts", 12345, 12345)

	// mariadb deploy
	mariadbEnv := []corev1.EnvVar{{
		Name:  "MYSQL_ROOT_PASSWORD",
		Value: "admin",
	}}

	// mariadb deployment & service
	r.reconcileDeploymentFor(context.TODO(), instance, "mariadb", instance.Spec.MariaDBImage, 3306, mariadbEnv, reqLogger)
	r.reconcileServiceFor(context.TODO(), instance, "mariadb", 3307, 3306)

	// keycloak env
	keycloakEnv := []corev1.EnvVar{{
		Name:  "KEYCLOAK_USER",
		Value: "admin",
	}, {
		Name:  "KEYCLOAK_PASSWORD",
		Value: "admin",
	}}

	// keycloak deployment & service
	r.reconcileDeploymentFor(context.TODO(), instance, "keycloak", instance.Spec.KeycloakImage, 8080, keycloakEnv, reqLogger)
	r.reconcileServiceFor(context.TODO(), instance, "keycloak", 8080, 8080)

	return reconcile.Result{}, nil
}

func (r *ReconcileRokkuSts) reconcileDeploymentForSts(ctx context.Context, s *rokkustsv1alpha1.RokkuSts, logger logr.Logger) error {
	serviceName := "rokkusts"
	deploymentName := types.NamespacedName{
		Name:      serviceName,
		Namespace: s.Namespace,
	}

	stsVars := []corev1.EnvVar{{
		Name:  "MARIADB_URL",
		Value: s.Spec.MariaDBUrl,
	}, {
		Name:  "STS_HOST",
		Value: "0.0.0.0",
	}, {
		Name:  "KEYCLOAK_URL",
		Value: s.Spec.KeycloakUrl,
	}, {
		Name:  "KEYCLOAK_CHECK_REALM_URL",
		Value: strconv.FormatBool(s.Spec.CheckRealm),
	}, {
		Name:  "KEYCLOAK_CHECK_ISSUER_FOR_LIST",
		Value: s.Spec.IssuedFor,
	}}

	newDeployment := r.genericDeployment(s, serviceName, s.Spec.StsImage, 12345, stsVars)
	stsFound := &appsv1.Deployment{}

	err := r.client.Get(ctx, deploymentName, stsFound)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a Deployment resource for: " + serviceName)

		return r.client.Create(ctx, newDeployment)
	}

	if err != nil {
		return fmt.Errorf("failed to retrieve Service resource: %v", err)
	}

	size := s.Spec.Size
	if *stsFound.Spec.Replicas != size {
		stsFound.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), stsFound)
		if err != nil {
			logger.Error(err, "Failed to update Deployment", "Deployment.Namespace", stsFound.Namespace, "Deployment.Name", stsFound.Name)
			return err
		}
		// Spec updated - return and requeue
		return nil
	}

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(s.Namespace),
		client.MatchingLabels(labelsFor(s.Name)),
	}
	if err = r.client.List(context.TODO(), podList, listOpts...); err != nil {
		logger.Error(err, "Failed to list pods", "RokkuSts.Namespace", s.Namespace, "RokkuSts.Name", s.Name)
		return err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, s.Status.Nodes) {
		s.Status.Nodes = podNames
		err := r.client.Status().Update(context.TODO(), s)
		if err != nil {
			logger.Error(err, "Failed to update RokkuSts status")
			return err
		}
	}

	return r.client.Update(ctx, newDeployment)
}

func (r *ReconcileRokkuSts) reconcileDeploymentFor(ctx context.Context, s *rokkustsv1alpha1.RokkuSts, serviceName string,
	imageName string, port int, envVars []corev1.EnvVar, logger logr.Logger) error {
	deploymentName := types.NamespacedName{
		Name:      serviceName,
		Namespace: s.Namespace,
	}
	//logger := log.WithName("reconcileService").WithValues("Deployment", deploymentName)

	newDeployment := r.genericDeployment(s, serviceName, imageName, port, envVars)

	currentDeployment := &appsv1.Deployment{}

	err := r.client.Get(ctx, deploymentName, currentDeployment)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a Deployment resource for: " + serviceName)

		return r.client.Create(ctx, newDeployment)
	}

	if err != nil {
		return fmt.Errorf("failed to retrieve Service resource: %v", err)
	}

	return r.client.Update(ctx, newDeployment)
}

func (r *ReconcileRokkuSts) genericDeployment(s *rokkustsv1alpha1.RokkuSts, serviceName string, imageName string, port int, envVars []corev1.EnvVar) *appsv1.Deployment {
	ls := labelsFor(serviceName)
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: s.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: imageName,
						Name:  serviceName,
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(port),
							Name:          serviceName,
						}},
						Env: envVars,
					}},
				},
			},
		},
	}
	controllerutil.SetControllerReference(s, dep, r.scheme)
	return dep
}

func (r *ReconcileRokkuSts) reconcileServiceFor(ctx context.Context, s *rokkustsv1alpha1.RokkuSts, serviceName string, port int, targetPort int) error {
	svcName := types.NamespacedName{
		Name:      fmt.Sprintf("%s-service", serviceName),
		Namespace: s.Namespace,
	}

	logger := log.WithName("reconcileService").WithValues("Service", svcName)

	newService := r.genericService(s, serviceName, port, targetPort)

	var currentService corev1.Service
	err := r.client.Get(ctx, svcName, &currentService)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a Service resource for: " + serviceName)

		return r.client.Create(ctx, newService)
	}

	if err != nil {
		return fmt.Errorf("failed to retrieve Service resource: %v", err)
	}

	//logger.Info("Updating Service resource for: "+serviceName)

	return r.client.Update(ctx, newService)
}

func (r *ReconcileRokkuSts) genericService(s *rokkustsv1alpha1.RokkuSts, serviceName string, port int, targetPort int) *corev1.Service {
	var fullServiceName = fmt.Sprintf("%s-service", serviceName)
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fullServiceName,
			Namespace: s.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(s, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    "RokkuSts",
				}),
			},
			Labels: labelsFor(serviceName),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(targetPort),
					Port:       int32(port),
				},
			},

			Selector: labelsFor(serviceName),
		},
	}
	controllerutil.SetControllerReference(s, service, r.scheme)
	return service
}

func labelsFor(name string) map[string]string {
	return map[string]string{"app": name, name + "_cr": name}
}

func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
