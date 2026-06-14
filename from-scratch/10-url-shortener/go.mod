module tour_of_go/projects/from-scratch/10-url-shortener

go 1.23.0

require (
	tour_of_go/projects/from-scratch/04-rate-limiter v0.0.0
	tour_of_go/projects/from-scratch/09-task-scheduler v0.0.0
)

replace (
	tour_of_go/projects/from-scratch/04-rate-limiter => ../04-rate-limiter
	tour_of_go/projects/from-scratch/06-message-queue => ../06-message-queue
	tour_of_go/projects/from-scratch/09-task-scheduler => ../09-task-scheduler
)
