source ~/.git-completion.bash
source ~/.git-prompt.sh
GIT_PS1_SHOWDIRTYSTATE=true
GIT_PS1_SHOWUNTRACKEDFILES=true
GIT_PS1_SHOWSTASHSTATE=true
GIT_PS1_SHOWUPSTREAM=auto
PS1="\[\e[1;32m\]\u@\h\[\e[m\]:\[\e[1;34m\]\W\[\e[m\]\[\e[33m\]\$(__git_ps1)\[\e[m\]\$ "
