sgip
====

SGIP protocol v1.2 (China Unicom).  It implements a bridge from TCP to HTTP.

SGIP is used to send or receive SMS between SP and SGP. SGIP is based on the TCP, but i like HTTP more than TCP. So this project implements a bridge from TCP to HTTP.
With this project, all the code in SP, except communicating between SP and SGP, would be based on HTTP.
