package controllers

import (
	"context"
	"fmt"
	"time"

	examplecomv1 "github.com/simran2104/Wordpress-Operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// WordpressReconciler reconciles a Wordpress object
type WordpressReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

var log = logf.Log.WithName("controller_wordpress")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Wordpress Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &WordpressReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("wordpress-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Wordpress
	err = c.Watch(&source.Kind{Type: &examplecomv1.Wordpress{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Wordpress

	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &examplecomv1.Wordpress{},
	})

	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &examplecomv1.Wordpress{},
	})

	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &examplecomv1.Wordpress{},
	})

	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that WordpressReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &WordpressReconciler{}

//+kubebuilder:rbac:groups=example.com,resources=wordpresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=example.com,resources=wordpresses/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=example.com,resources=wordpresses/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Wordpress object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile

func (r *WordpressReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Wordpress")

	// Fetch the Wordpress instance
	//	instance := &examplecomv1.Wordpress{}
	wordpress := &examplecomv1.Wordpress{}
	err := r.Client.Get(context.TODO(), request.NamespacedName, wordpress)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	var result *reconcile.Result

	// === MYSQL ======

	result, err = r.ensurePVC(request, wordpress, r.pvcForMysql(wordpress))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureDeployment(request, wordpress, r.deploymentForMysql(wordpress))
	if result != nil {
		return *result, err
	}
	result, err = r.ensureService(request, wordpress, r.serviceForMysql(wordpress))
	if result != nil {
		return *result, err
	}

	mysqlRunning := r.isMysqlUp(wordpress)

	if !mysqlRunning {
		// If MySQL isn't running yet, requeue the reconcile
		// to run again after a delay
		delay := time.Second * time.Duration(5)

		log.Info(fmt.Sprintf("MySQL isn't running, waiting for %s", delay))
		return reconcile.Result{RequeueAfter: delay}, nil
	}

	// ===== WORDPRESS =====

	result, err = r.ensurePVC(request, wordpress, r.pvcForWordpress(wordpress))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureDeployment(request, wordpress, r.deploymentForWordpress(wordpress))
	if result != nil {
		return *result, err
	}

	result, err = r.ensureService(request, wordpress, r.serviceForWordpress(wordpress))
	if result != nil {
		return *result, err
	}

	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WordpressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplecomv1.Wordpress{}).
		Complete(r)
}
