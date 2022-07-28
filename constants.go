package npmstart

const (
	Node        = "node"
	NodeModules = "node_modules"
	Npm         = "npm"
)

const StartupScript = `trap 'kill -TERM $CPID' TERM
trap 'kill -INT $CPID' INT
( %s ) &
CPID="$!"
wait $CPID`
