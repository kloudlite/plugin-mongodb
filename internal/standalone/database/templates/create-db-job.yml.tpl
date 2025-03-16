apiVersion: batch/v1
kind: Job
metadata: {{.JobMetadata | toJson }}
spec:
  template:
    metadata:
      labels: {{.PodLabels | toJson }}
      annotations: {{.PodAnnotations | toJson }}
    spec:
      tolerations: {{.Tolerations | toJson }}
      nodeSelector: {{.NodeSelector | toJson }}

      restartPolicy: "OnFailure"

      containers:
        - name: mongodb
          image: mongo:latest
          imagePullPolicy: Always

          resources:
            requests:
              cpu: 100m
              memory: 100Mi
            limits:
              cpu: 200m
              memory: 200Mi

          env: &env
            - name: MONGODB_URI
              valueFrom:
                secretKeyRef:
                  name: {{.RootUserCredentialsSecret}}
                  key: .CLUSTER_LOCAL_URI

            - name: NEW_DB_NAME
              valueFrom:
                secretKeyRef:
                  name: {{.NewUserCredentialsSecret}}
                  key: DB_NAME

            - name: NEW_USERNAME
              valueFrom:
                secretKeyRef:
                  name: {{.NewUserCredentialsSecret}}
                  key: USERNAME

            - name: NEW_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{.NewUserCredentialsSecret}}
                  key: PASSWORD

          command:
          - sh
          - -c
          - |+
            cat > /tmp/mongoscript.js <<EOF
            use $NEW_DB_NAME;

            if (db.getUser("$NEW_USERNAME") == null) {
              db.createUser({
                "user": "$NEW_USERNAME",
                "pwd": "$NEW_PASSWORD",
                "roles": [
                  {
                    "role": "readWrite",
                    "db": "$NEW_DB_NAME"
                  },
                  {
                    "role": "dbAdmin",
                    "db": "$NEW_DB_NAME"
                  }
                ]
              })
            } else {
              db.changeUserPassword("$NEW_USERNAME", "$NEW_PASSWORD")
            } 


            EOF
            
            mongosh "$MONGODB_URI" < /tmp/mongoscript.js

