package controllers

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appv1 "github.com/breuerfelix/medusa/operator/api/v1"
)

const (
	FINALIZER = "medusa.fbr.ai/operator"
)

// MedusaReconciler reconciles a Medusa object
type MedusaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	DBURL  string
}

//+kubebuilder:rbac:groups=app.medusa.fbr.ai,resources=medusas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.medusa.fbr.ai,resources=medusas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.medusa.fbr.ai,resources=medusas/finalizers,verbs=update

func (r *MedusaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	medusa := &appv1.Medusa{}
	if err := r.Client.Get(ctx, req.NamespacedName, medusa); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// deletion routine
	if !medusa.DeletionTimestamp.IsZero() {
		for _, prefix := range []string{"m-", "a-"} {
			if err := deleteDeployment(
				ctx, r.Client,
				metav1.ObjectMeta{
					Namespace: medusa.Namespace,
					Name:      prefix + medusa.Name,
				},
			); err != nil {
				return ctrl.Result{}, err
			}

			if err := deleteService(
				ctx, r.Client,
				metav1.ObjectMeta{
					Namespace: medusa.Namespace,
					Name:      prefix + medusa.Name,
				},
			); err != nil {
				return ctrl.Result{}, err
			}

			if err := deleteIngress(
				ctx, r.Client,
				metav1.ObjectMeta{
					Namespace: medusa.Namespace,
					Name:      prefix + medusa.Name,
				},
			); err != nil {
				return ctrl.Result{}, err
			}
		}

		// TODO delete secret for database / delete database ?

		// delete finalizer if exists
		if controllerutil.ContainsFinalizer(medusa, FINALIZER) {
			if _, err := controllerutil.CreateOrUpdate(
				ctx, r.Client, medusa, func() error {
					controllerutil.RemoveFinalizer(medusa, FINALIZER)
					return nil
				},
			); err != nil {
				return ctrl.Result{}, err
			}
		}

		// return because we are in deletion routine
		return ctrl.Result{}, nil
	}

	// add finalizer if not exists
	if !controllerutil.ContainsFinalizer(medusa, FINALIZER) {
		if _, err := controllerutil.CreateOrUpdate(
			ctx, r.Client, medusa, func() error {
				controllerutil.AddFinalizer(medusa, FINALIZER)
				return nil
			},
		); err != nil {
			return ctrl.Result{}, err
		}
	}

	// create postgres user
	db, err := sql.Open("postgres", r.DBURL)
	if err != nil {
		return ctrl.Result{}, err
	}

	database := medusa.Name
	user := medusa.Name
	// TODO generate password
	password := medusa.Name

	// TODO only execute if not already exists
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE "%s";`, database)); err != nil {
		fmt.Println("create database error")
		fmt.Println(err)
	}

	if _, err := db.ExecContext(ctx, fmt.Sprintf(`CREATE USER "%s";`, user)); err != nil {
		fmt.Println("create user error")
		fmt.Println(err)
	}

	if _, err := db.ExecContext(ctx, fmt.Sprintf(
		`ALTER USER "%s" PASSWORD '%s'; GRANT ALL PRIVILEGES ON DATABASE "%s" TO "%s";`,
		user, password, database, user)); err != nil {
		fmt.Println("altering error")
		fmt.Println(err)
	}

	db.Close()

	// medusa backend
	if err := createOrUpdateDeployment(
		ctx,
		r.Client,
		medusa.Namespace,
		"m-"+medusa.Name,
		medusa.Spec.Backend.Image,
		9000,
		[]corev1.EnvVar{
			{
				Name:  "ADMIN_CORS",
				Value: "https://" + medusa.Spec.Admin.Domain,
			},
			{
				Name: "DATABASE_URL",
				// TODO create secret for that
				Value: fmt.Sprintf("postgresql://%s:%s@postgres-postgresql.postgres:5432/%s", user, password, database),
			},
			{
				Name:  "REDIS_URL",
				Value: medusa.Spec.Backend.RedisURL,
			},
		},
	); err != nil {
		return ctrl.Result{}, err
	}

	// medusa backend service
	if err := createOrUpdateService(
		ctx,
		r.Client,
		medusa.Namespace,
		"m-"+medusa.Name,
	); err != nil {
		return ctrl.Result{}, err
	}

	// medusa backend ingress
	if err := createOrUpdateIngress(
		ctx,
		r.Client,
		medusa.Namespace,
		"m-"+medusa.Name,
		medusa.Spec.Backend.Domain,
	); err != nil {
		return ctrl.Result{}, err
	}

	// medusa admin
	if err := createOrUpdateDeployment(
		ctx,
		r.Client,
		medusa.Namespace,
		"a-"+medusa.Name,
		medusa.Spec.Admin.Image,
		80,
		[]corev1.EnvVar{
			{
				Name:  "MEDUSA_URL",
				Value: "https://" + medusa.Spec.Backend.Domain,
			},
		},
	); err != nil {
		return ctrl.Result{}, err
	}

	// medusa admin service
	if err := createOrUpdateService(
		ctx,
		r.Client,
		medusa.Namespace,
		"a-"+medusa.Name,
	); err != nil {
		return ctrl.Result{}, err
	}

	// medusa admin ingress
	if err := createOrUpdateIngress(
		ctx,
		r.Client,
		medusa.Namespace,
		"a-"+medusa.Name,
		medusa.Spec.Admin.Domain,
	); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func deleteDeployment(
	ctx context.Context, c client.Client, meta metav1.ObjectMeta,
) error {
	return client.IgnoreNotFound(c.Delete(ctx, &appsv1.Deployment{
		ObjectMeta: meta,
	}))
}

func deleteService(
	ctx context.Context, c client.Client, meta metav1.ObjectMeta,
) error {
	return client.IgnoreNotFound(c.Delete(ctx, &corev1.Service{
		ObjectMeta: meta,
	}))
}

func deleteIngress(
	ctx context.Context, c client.Client, meta metav1.ObjectMeta,
) error {
	return client.IgnoreNotFound(c.Delete(ctx, &networkingv1.Ingress{
		ObjectMeta: meta,
	}))
}

func createOrUpdateIngress(
	ctx context.Context,
	client client.Client,
	namespace string,
	name string,
	domain string,
) error {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, ingress, func() error {
		ingress.Annotations = map[string]string{
			"cert-manager.io/cluster-issuer": "letsencrypt",
			"kubernetes.io/ingress.class":    "traefik",
		}
		prefix := networkingv1.PathTypePrefix
		ingress.Spec = networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: domain,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: name,
									Port: networkingv1.ServiceBackendPort{
										Number: 80,
									},
								},
							},
							Path:     "/",
							PathType: &prefix,
						}},
					},
				},
			}},
			TLS: []networkingv1.IngressTLS{{
				Hosts:      []string{domain},
				SecretName: name + "-tls",
			}},
		}
		return nil
	})

	return err
}

func createOrUpdateService(
	ctx context.Context,
	client client.Client,
	namespace string,
	name string,
) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, service, func() error {
		service.Spec = corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Name:       "http",
				TargetPort: intstr.FromString("http"),
				Port:       80,
				Protocol:   corev1.ProtocolTCP,
			}},
			Selector: map[string]string{
				"name": name,
			},
		}
		return nil
	})

	return err
}

func createOrUpdateDeployment(
	ctx context.Context,
	client client.Client,
	namespace string,
	name string,
	image string,
	port int32,
	env []corev1.EnvVar,
) error {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, client, deployment, func() error {
		deployment.Spec = getDeploymentSpec(
			name,
			image,
			port,
			env,
		)
		return nil
	})

	return err
}

func getDeploymentSpec(
	name string,
	image string,
	port int32,
	env []corev1.EnvVar,
) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"name": name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"name": name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:            name,
					Image:           image,
					ImagePullPolicy: "Always",
					Ports: []corev1.ContainerPort{{
						Name:          "http",
						ContainerPort: port,
						Protocol:      corev1.ProtocolTCP,
					}},
					Env: env,
				}},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MedusaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.Medusa{}).
		Complete(r)
}
