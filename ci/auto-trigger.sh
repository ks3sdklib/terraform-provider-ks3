international_regions=(eu-central-1)
international_time_location=(Europe/London Asia/Kolkata)

for (( i=0;i<${#international_regions[@]};i++)) do
  fly -t terraform-intl sp -p auto-trigger \
    -c auto-trigger.yml \
    -v ksyun_cli_bucket=$ALIYUN_CLI_BUCKET \
    -v ksyun_cli_region=$ALIYUN_CLI_REGION \
    -v ksyun_cli_access_key=$ALIYUN_CLI_ACCESS_KEY \
    -v ksyun_cli_secret_key=$ALIYUN_CLI_SECRET_KEY \
    -v ksyun_account_site="Domestic" \
    -v access_ci_url=$INTL_ACCESS_CI_URL \
    -v access_ci_user_name=$INTL_ACCESS_CI_USER_NAME \
    -v access_ci_password=$INTL_ACCESS_CI_PASSWORD \
    -v ding_talk_token=$DING_TALK_TOKEN \
    -v ksyun_accound_id=$ALICLOUD_ACCOUNT_ID_MASTER \
    -v ksyun_resource_group_id="" \
    -v ksyun_waf_instance_id=$ALICLOUD_WAF_INSTANCE_ID \
    -v time_location=${international_time_location[i]} \
    -v ksyun_access_key=$ALICLOUD_ACCESS_KEY_MASTER \
    -v ksyun_secret_key=$ALICLOUD_SECRET_KEY_MASTER \
    -v ksyun_region=${international_regions[i]} \
    -v ksyun_concourse_target=terraform-china \
    -v ksyun_concourse_target_url=$CHINA_ACCESS_CI_URL \
    -v ksyun_concourse_target_user=$CHINA_ACCESS_CI_USER_NAME \
    -v ksyun_concourse_target_password=$CHINA_ACCESS_CI_PASSWORD \
    -v ksyun_concourse_target_pipeline_name="cn-hangzhou" \
    -v ksyun_trigger_target_pipeline=false \
    -v ksyun_access_key_master=$ALICLOUD_ACCESS_KEY_MASTER \
    -v ksyun_secret_key_master=$ALICLOUD_SECRET_KEY_MASTER \
    -v ksyun_access_key_slave=$ALICLOUD_ACCESS_KEY_SLAVE \
    -v ksyun_secret_key_slave=$ALICLOUD_SECRET_KEY_SLAVE \
    -v enterprise_account_enabled=true
done
