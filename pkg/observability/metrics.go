package observability

























}	prometheus.MustRegister(CourtReviewsTotal, SandboxStartsTotal)func init() {)	// Add more metrics	)		},			Help: "Total sandbox starts",			Name: "sandbox_starts_total",		prometheus.CounterOpts{	SandboxStartsTotal = prometheus.NewCounter(	)		[]string{"result"},		},			Help: "Total number of court reviews",			Name: "court_reviews_total",		prometheus.CounterOpts{	CourtReviewsTotal = prometheus.NewCounterVec(var ()	"github.com/prometheus/client_golang/prometheus"import (package observability