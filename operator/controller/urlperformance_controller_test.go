package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"

	traefikofficerv1alpha1 "github.com/mithucste30/traefik-officer-operator/operator/api/v1alpha1"
)

var _ = Describe("UrlPerformance Reconciler", func() {

	var (
		ctx                 context.Context
		configManager       *ConfigManager
		reconciler          *UrlPerformanceReconciler
		testNamespace       string
		testUrlPerformance  *traefikofficerv1alpha1.UrlPerformance
		testIngress         *networkingv1.Ingress
	)

	BeforeEach(func() {
		ctx = context.Background()
		configManager = createTestConfigManager()
		testNamespace = "default"

		reconciler = &UrlPerformanceReconciler{
			Client:        k8sClient,
			Scheme:        scheme.Scheme,
			ConfigManager: configManager,
		}
	})

	AfterEach(func() {
		// Cleanup test resources
		if testUrlPerformance != nil {
			_ = k8sClient.Delete(ctx, testUrlPerformance)
		}
		if testIngress != nil {
			_ = k8sClient.Delete(ctx, testIngress)
		}
	})

	Context("Scenario A: Target Ingress exists and is valid", func() {
		It("should successfully create UrlPerformance resource and update config", func() {
			By("creating a test Ingress")
			testIngress = &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress-a",
					Namespace: testNamespace,
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "test.example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path: "/api",
							PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "test-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testIngress)).To(Succeed())

			By("creating a UrlPerformance resource")
			testUrlPerformance = &traefikofficerv1alpha1.UrlPerformance{
				TypeMeta: metav1.TypeMeta{
					Kind:       "UrlPerformance",
					APIVersion: "traefikofficer.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-urlperf-a",
					Namespace: testNamespace,
				},
				Spec: traefikofficerv1alpha1.UrlPerformanceSpec{
					TargetRef: traefikofficerv1alpha1.TargetReference{
						Kind:     "Ingress",
						Name:     testIngress.Name,
						Namespace: testNamespace,
					},
					WhitelistPathsRegex: []string{"/api/.*"},
					CollectNTop:         20,
					Enabled:             true,
				},
			}
			GinkgoWriter.Printf("DEBUG: About to create UrlPerformance: Kind=%s, APIVersion=%s, Name=%s\n",
				testUrlPerformance.Kind, testUrlPerformance.APIVersion, testUrlPerformance.Name)
			Expect(k8sClient.Create(ctx, testUrlPerformance)).To(Succeed())

			By("reconciling the resource")
			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			By("verifying the status was updated")
			Eventually(func() bool {
				urlPerf := &traefikofficerv1alpha1.UrlPerformance{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				}, urlPerf)
				if err != nil {
					return false
				}
				return urlPerf.Status.Phase == traefikofficerv1alpha1.PhaseActive
			}, timeout, interval).Should(BeTrue())

			By("verifying config was updated in ConfigManager")
			configKey := testNamespace + "-" + testIngress.Name
			config, exists := configManager.GetConfig(configKey)
			Expect(exists).To(BeTrue())
			Expect(config.Enabled).To(BeTrue())
			Expect(config.TargetName).To(Equal(testIngress.Name))
		})
	})

	Context("Scenario B: Target Ingress does not exist", func() {
		It("should set error status when target Ingress is missing", func() {
			By("creating a UrlPerformance resource referencing non-existent Ingress")
			testUrlPerformance = &traefikofficerv1alpha1.UrlPerformance{
				TypeMeta: metav1.TypeMeta{
					Kind:       "UrlPerformance",
					APIVersion: "traefikofficer.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-urlperf-b",
					Namespace: testNamespace,
				},
				Spec: traefikofficerv1alpha1.UrlPerformanceSpec{
					TargetRef: traefikofficerv1alpha1.TargetReference{
						Kind:     "Ingress",
						Name:     "non-existent-ingress",
						Namespace: testNamespace,
					},
					CollectNTop: 20,
					Enabled:     true,
				},
			}
			Expect(k8sClient.Create(ctx, testUrlPerformance)).To(Succeed())

			By("reconciling the resource")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the status shows error phase")
			Eventually(func() bool {
				urlPerf := &traefikofficerv1alpha1.UrlPerformance{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				}, urlPerf)
				if err != nil {
					return false
				}
				return urlPerf.Status.Phase == traefikofficerv1alpha1.PhaseError
			}, timeout, interval).Should(BeTrue())

			By("verifying TargetExists condition is False")
			urlPerf := &traefikofficerv1alpha1.UrlPerformance{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				}, urlPerf)
				if err != nil {
					return false
				}
				for _, cond := range urlPerf.Status.Conditions {
					if string(cond.Type) == "TargetExists" {
						return cond.Status == "False"
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Scenario C: UrlPerformance resource is disabled", func() {
		It("should remove config when UrlPerformance is disabled", func() {
			By("creating a test Ingress")
			testIngress = &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress-c",
					Namespace: testNamespace,
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "test.example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path: "/api",
							PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "test-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testIngress)).To(Succeed())

			By("creating a disabled UrlPerformance resource")
			testUrlPerformance = &traefikofficerv1alpha1.UrlPerformance{
				TypeMeta: metav1.TypeMeta{
					Kind:       "UrlPerformance",
					APIVersion: "traefikofficer.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-urlperf-c",
					Namespace: testNamespace,
				},
				Spec: traefikofficerv1alpha1.UrlPerformanceSpec{
					TargetRef: traefikofficerv1alpha1.TargetReference{
						Kind:     "Ingress",
						Name:     testIngress.Name,
						Namespace: testNamespace,
					},
					CollectNTop: 20,
					Enabled:     false,
				},
			}
			Expect(k8sClient.Create(ctx, testUrlPerformance)).To(Succeed())

			By("reconciling the resource")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the status shows Disabled phase")
			Eventually(func() bool {
				urlPerf := &traefikofficerv1alpha1.UrlPerformance{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				}, urlPerf)
				if err != nil {
					return false
				}
				return urlPerf.Status.Phase == traefikofficerv1alpha1.PhaseDisabled
			}, timeout, interval).Should(BeTrue())

			By("verifying config was removed from ConfigManager")
			configKey := testNamespace + "-" + testIngress.Name
			_, exists := configManager.GetConfig(configKey)
			Expect(exists).To(BeFalse())
		})
	})

	Context("Scenario D: Invalid whitelist regex", func() {
		It("should set error status when whitelist regex is invalid", func() {
			By("creating a test Ingress")
			testIngress = &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress-d",
					Namespace: testNamespace,
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "test.example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path: "/api",
							PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "test-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testIngress)).To(Succeed())

			By("creating a UrlPerformance resource with invalid regex")
			testUrlPerformance = &traefikofficerv1alpha1.UrlPerformance{
				TypeMeta: metav1.TypeMeta{
					Kind:       "UrlPerformance",
					APIVersion: "traefikofficer.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-urlperf-d",
					Namespace: testNamespace,
				},
				Spec: traefikofficerv1alpha1.UrlPerformanceSpec{
					TargetRef: traefikofficerv1alpha1.TargetReference{
						Kind:     "Ingress",
						Name:     testIngress.Name,
						Namespace: testNamespace,
					},
					WhitelistPathsRegex: []string{"[invalid(regex"},
					CollectNTop:         20,
					Enabled:             true,
				},
			}
			Expect(k8sClient.Create(ctx, testUrlPerformance)).To(Succeed())

			By("reconciling the resource")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the status shows error phase")
			Eventually(func() bool {
				urlPerf := &traefikofficerv1alpha1.UrlPerformance{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				}, urlPerf)
				if err != nil {
					return false
				}
				return urlPerf.Status.Phase == traefikofficerv1alpha1.PhaseError
			}, timeout, interval).Should(BeTrue())

			By("verifying ConfigGenerated condition is False with InvalidRegex reason")
			urlPerf := &traefikofficerv1alpha1.UrlPerformance{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				}, urlPerf)
				if err != nil {
					return false
				}
				for _, cond := range urlPerf.Status.Conditions {
					if string(cond.Type) == "ConfigGenerated" {
						return cond.Status == "False" && cond.Reason == "InvalidRegex"
					}
				}
				return false
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Scenario E: Invalid ignored regex", func() {
		It("should set error status when ignored regex is invalid", func() {
			By("creating a test Ingress")
			testIngress = &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress-e",
					Namespace: testNamespace,
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "test.example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path: "/api",
							PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "test-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testIngress)).To(Succeed())

			By("creating a UrlPerformance resource with invalid ignored regex")
			testUrlPerformance = &traefikofficerv1alpha1.UrlPerformance{
				TypeMeta: metav1.TypeMeta{
					Kind:       "UrlPerformance",
					APIVersion: "traefikofficer.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-urlperf-e",
					Namespace: testNamespace,
				},
				Spec: traefikofficerv1alpha1.UrlPerformanceSpec{
					TargetRef: traefikofficerv1alpha1.TargetReference{
						Kind:     "Ingress",
						Name:     testIngress.Name,
						Namespace: testNamespace,
					},
					IgnoredPathsRegex: []string{"(?P<invalid"},
					CollectNTop:       20,
					Enabled:           true,
				},
			}
			Expect(k8sClient.Create(ctx, testUrlPerformance)).To(Succeed())

			By("reconciling the resource")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the status shows error phase")
			Eventually(func() bool {
				urlPerf := &traefikofficerv1alpha1.UrlPerformance{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				}, urlPerf)
				if err != nil {
					return false
				}
				return urlPerf.Status.Phase == traefikofficerv1alpha1.PhaseError
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Scenario F: Multiple UrlPerformance resources", func() {
		It("should handle multiple UrlPerformance resources correctly", func() {
			By("creating two test Ingresses")
			ingress1 := &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress-f1",
					Namespace: testNamespace,
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "test1.example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path: "/api1",
								PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "test-service1",
													Port: networkingv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, ingress1)).To(Succeed())
			defer k8sClient.Delete(ctx, ingress1)

			ingress2 := &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress-f2",
					Namespace: testNamespace,
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "test2.example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path: "/api2",
								PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "test-service2",
													Port: networkingv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, ingress2)).To(Succeed())
			defer k8sClient.Delete(ctx, ingress2)

			By("creating two UrlPerformance resources")
			urlPerf1 := &traefikofficerv1alpha1.UrlPerformance{
				TypeMeta: metav1.TypeMeta{
					Kind:       "UrlPerformance",
					APIVersion: "traefikofficer.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-urlperf-f1",
					Namespace: testNamespace,
				},
				Spec: traefikofficerv1alpha1.UrlPerformanceSpec{
					TargetRef: traefikofficerv1alpha1.TargetReference{
						Kind:     "Ingress",
						Name:     ingress1.Name,
						Namespace: testNamespace,
					},
					CollectNTop: 20,
					Enabled:     true,
				},
			}
			Expect(k8sClient.Create(ctx, urlPerf1)).To(Succeed())
			defer k8sClient.Delete(ctx, urlPerf1)

			urlPerf2 := &traefikofficerv1alpha1.UrlPerformance{
				TypeMeta: metav1.TypeMeta{
					Kind:       "UrlPerformance",
					APIVersion: "traefikofficer.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-urlperf-f2",
					Namespace: testNamespace,
				},
				Spec: traefikofficerv1alpha1.UrlPerformanceSpec{
					TargetRef: traefikofficerv1alpha1.TargetReference{
						Kind:     "Ingress",
						Name:     ingress2.Name,
						Namespace: testNamespace,
					},
					CollectNTop: 20,
					Enabled:     true,
				},
			}
			Expect(k8sClient.Create(ctx, urlPerf2)).To(Succeed())
			defer k8sClient.Delete(ctx, urlPerf2)

			By("reconciling both resources")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: urlPerf1.Namespace,
					Name:      urlPerf1.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: urlPerf2.Namespace,
					Name:      urlPerf2.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying both resources are active")
			Eventually(func() bool {
				up1 := &traefikofficerv1alpha1.UrlPerformance{}
				err1 := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: urlPerf1.Namespace,
					Name:      urlPerf1.Name,
				}, up1)

				up2 := &traefikofficerv1alpha1.UrlPerformance{}
				err2 := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: urlPerf2.Namespace,
					Name:      urlPerf2.Name,
				}, up2)

				return err1 == nil && err2 == nil &&
					up1.Status.Phase == traefikofficerv1alpha1.PhaseActive &&
					up2.Status.Phase == traefikofficerv1alpha1.PhaseActive
			}, timeout, interval).Should(BeTrue())

			By("verifying both configs are in ConfigManager")
			configs := configManager.GetAllConfigs()
			Expect(len(configs)).To(Equal(2))
		})
	})

	Context("Scenario G: Updating UrlPerformance resource", func() {
		It("should handle updates to UrlPerformance resources", func() {
			By("creating a test Ingress")
			testIngress = &networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress-g",
					Namespace: testNamespace,
				},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: "test.example.com",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path: "/api",
							PathType: func() *networkingv1.PathType { pt := networkingv1.PathTypePrefix; return &pt }(),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: "test-service",
													Port: networkingv1.ServiceBackendPort{
														Number: 80,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testIngress)).To(Succeed())

			By("creating a UrlPerformance resource")
			testUrlPerformance = &traefikofficerv1alpha1.UrlPerformance{
				TypeMeta: metav1.TypeMeta{
					Kind:       "UrlPerformance",
					APIVersion: "traefikofficer.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-urlperf-g",
					Namespace: testNamespace,
				},
				Spec: traefikofficerv1alpha1.UrlPerformanceSpec{
					TargetRef: traefikofficerv1alpha1.TargetReference{
						Kind:     "Ingress",
						Name:     testIngress.Name,
						Namespace: testNamespace,
					},
					WhitelistPathsRegex: []string{"/api/.*"},
					CollectNTop:         20,
					Enabled:             true,
				},
			}
			Expect(k8sClient.Create(ctx, testUrlPerformance)).To(Succeed())

			By("reconciling the resource")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying initial state")
			Eventually(func() bool {
				urlPerf := &traefikofficerv1alpha1.UrlPerformance{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				}, urlPerf)
				if err != nil {
					return false
				}
				return urlPerf.Status.Phase == traefikofficerv1alpha1.PhaseActive
			}, timeout, interval).Should(BeTrue())

			By("updating the UrlPerformance resource")
			Eventually(func() error {
				urlPerf := &traefikofficerv1alpha1.UrlPerformance{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				}, urlPerf)
				if err != nil {
					return err
				}
				urlPerf.Spec.CollectNTop = 50
				urlPerf.Spec.IgnoredPathsRegex = []string{"/healthz"}
				return k8sClient.Update(ctx, urlPerf)
			}, timeout, interval).Should(Succeed())

			By("reconciling the updated resource")
			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: testUrlPerformance.Namespace,
					Name:      testUrlPerformance.Name,
				},
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying config was updated in ConfigManager")
			configKey := testNamespace + "-" + testIngress.Name
			Eventually(func() bool {
				config, exists := configManager.GetConfig(configKey)
				return exists && config.CollectNTop == 50 &&
					len(config.IgnoredRegex) == 1
			}, timeout, interval).Should(BeTrue())
		})
	})
})

const (
	timeout = 5 * time.Second
	interval = 250 * time.Millisecond
)
