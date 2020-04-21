package rokkusts

import (
	"context"
	"k8s.io/api/auditregistration/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"reflect"

	rokkustsv1alpha1 "github.com/arempter/sts-operator/pkg/apis/rokkusts/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRokkuSts) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RokkuSts")

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

	stsFound := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, stsFound)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForSts(instance)
		reqLogger.Info("Creating a new RokkuSts Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new RokkuSts Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get RokkuSts Deployment")
		return reconcile.Result{}, err
	}

	// mariadb deploy
	mariadbFound := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "mariadb", Namespace: instance.Namespace}, mariadbFound)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForMariaDb(instance)
		reqLogger.Info("Creating a new MariaDb Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new MariaDb Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get MariaDb Deployment")
		return reconcile.Result{}, err
	}
	// mariadb service
	serviceFound := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "mariadb-service", Namespace: instance.Namespace}, serviceFound)
	if err != nil && errors.IsNotFound(err) {
		// Define a new svc
		svc := r.MariaDBService(instance)
		reqLogger.Info("Creating a new mariadb service", "Deployment.Namespace", svc.Namespace, "Deployment.Name", svc.Name)
		err = r.client.Create(context.TODO(), svc)
		if err != nil {
			reqLogger.Error(err, "Failed to create new mariadb service", "Deployment.Namespace", svc.Namespace, "Deployment.Name", svc.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get mariadb service")
		return reconcile.Result{}, err
	}

	// keycloak deploy
	keycloakFound := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "keycloak", Namespace: instance.Namespace}, keycloakFound)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForKeycloak(instance)
		reqLogger.Info("Creating a new keycloak Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new keycloak Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get keycloak Deployment")
		return reconcile.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	size := instance.Spec.Size
	if *stsFound.Spec.Replicas != size {
		stsFound.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), stsFound)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment", "Deployment.Namespace", stsFound.Namespace, "Deployment.Name", stsFound.Name)
			return reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Update the rokku-sts status with the pod names
	// List the pods for this deployment
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.Namespace),
		client.MatchingLabels(labelsForSts(instance.Name)),
	}
	if err = r.client.List(context.TODO(), podList, listOpts...); err != nil {
		reqLogger.Error(err, "Failed to list pods", "RokkuSts.Namespace", instance.Namespace, "RokkuSts.Name", instance.Name)
		return reconcile.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, instance.Status.Nodes) {
		instance.Status.Nodes = podNames
		err := r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update RokkuSts status")
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileRokkuSts) deploymentForMariaDb(s *rokkustsv1alpha1.RokkuSts) *appsv1.Deployment {
	ls := labelsForMaria("mariadb")
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mariadb",
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
						Image: "wbaa/rokku-dev-mariadb:0.0.8",
						Name:  "mariadb",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 3307,
							Name:          "mariadb",
						}},
						Env: []corev1.EnvVar{{
							Name:  "MYSQL_ROOT_PASSWORD",
							Value: "admin",
						}},
					}},
				},
			},
		},
	}
	controllerutil.SetControllerReference(s, dep, r.scheme)
	return dep
}
func (r *ReconcileRokkuSts) MariaDBService(s *rokkustsv1alpha1.RokkuSts) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mariadb-service",
			Namespace: s.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(s, schema.GroupVersionKind{
					Group:   v1alpha1.SchemeGroupVersion.Group,
					Version: v1alpha1.SchemeGroupVersion.Version,
					Kind:    "RokkuSts",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("http"),
					Port:       int32(3307),
				},
			},
			Selector: labelsForMaria("mariadb"),
		},
	}
	return service
}

func (r *ReconcileRokkuSts) deploymentForKeycloak(s *rokkustsv1alpha1.RokkuSts) *appsv1.Deployment {
	ls := labelsForKeyclok("keycloak")
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keycloak",
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
						Image: "wbaa/rokku-dev-keycloak:0.0.6",
						Name:  "keycloak",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 8080,
							Name:          "keycloak",
						}},
						//todo: add password
						Env: []corev1.EnvVar{{
							Name:  "KEYCLOAK_USER",
							Value: "admin",
						}, {
							Name:  "KEYCLOAK_PASSWORD",
							Value: "admin",
						}},
					}},
				},
			},
		},
	}
	controllerutil.SetControllerReference(s, dep, r.scheme)
	return dep
}

func (r *ReconcileRokkuSts) deploymentForSts(s *rokkustsv1alpha1.RokkuSts) *appsv1.Deployment {
	ls := labelsForSts(s.Name)
	replicas := s.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "wbaa/rokku-sts:0.3.4",
						Name:  "rokku-sts",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 12345,
							Name:          "rokku-sts",
						}},
						Env: []corev1.EnvVar{{
							Name:  "MARIADB_URL",
							Value: "jdbc:mysql:loadbalance://mariadb-service:3307,mariadb-service:3307/rokku",
						}},
					}},
				},
			},
		},
	}
	// Set rokku-sts instance as the owner and controller
	controllerutil.SetControllerReference(s, dep, r.scheme)
	return dep

}

func labelsForSts(name string) map[string]string {
	return map[string]string{"app": "rokkusts", "rokku-sts_cr": name}
}
func labelsForMaria(name string) map[string]string {
	return map[string]string{"app": "mariadb", "mariadb_cr": name}
}
func labelsForKeyclok(name string) map[string]string {
	return map[string]string{"app": "keycloak", "keycloak_cr": name}
}

func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
