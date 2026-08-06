select
  a.contract_inst,
  a.scene,
  b.match_rule,
  c.charge_rule_content,
  d.collect_fee_type,
  d.collect_fee_mode
from
  (
    select
      *
    from
      bytepay_charge_contract
    where
      contract_inst = '{{contract_inst}}'
  ) a
  LEFT JOIN (
    select
      *
    from
      bytepay_charge_factor
  ) b on a.charge_contract_code = b.charge_contract_code
  LEFT JOIN (
    select
      *
    from
      charge_rule_info
  ) c on b.charge_rule_info_id = c.charge_rule_info_id
  LEFT JOIN (
    SELECT
      *
    from
      collect_fee_rule_info
  ) d on b.collect_fee_rule_info_id = d.collect_fee_rule_info_id;
