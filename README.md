cf-redis-broker
===============

## Integration Tests
### AWS Keys
The backup integration tests send data to AWS S3, and need the following
evironment variables set to be able to work:

* `AWS_ACCESS_KEY_ID`
* `AWS_SECRET_ACCESS_KEY`

These are defined in `integration/backup/assets/working-backup.yml.template`.

It also expect a bucket called `redis-backup-test`, and the user whose key is
provided, above, should have the following access to the bucket for testing
purposes:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "RestrictiveBackups",
            "Effect": "Allow",
            "Action": [
                "s3:CreateBucket",
                "s3:PutObject",
                "s3:DeleteObject",
                "s3:ListObject",
                "s3:ListBucket"
            ],
            "Resource": [
                "arn:aws:s3:::redis-backup-test",
                "arn:aws:s3:::redis-backup-test/*"
            ]
        },
    ]
}
```
