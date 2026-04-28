#!/usr/bin/env bash

ARGS='--model=gpt-5-mini --provider=github-copilot "tell me a joke"' make run
cat <<EOF

==============================================================================

EOF

ARGS='--model=deepseek-v4-pro --provider=deepseek "tell me a joke"' make run

cat <<EOF

==============================================================================

EOF

ARGS='--model=deepseek/deepseek-v4-flash "tell me a joke"' make run
