// СГЕНЕРИРОВАНО contracts/scripts/gen-ts.mjs (make generate) — руками не править (AD-20).
// Источник: contracts/constants.yaml

export const NS_ANT = "4b82fbf1-fc9e-5a06-91c6-8c100ebac4ef" as const
export const BPMN_EXT_URI = "urn:ant:bpmn-ext:1" as const
export const BPMN_EXT_PREFIX = "ant" as const
export const PAYLOAD_TYPE_TEMPLATE = "application/vnd.ant.{class}+json; v={version}" as const
export const MLDSA_CONTEXT_TEMPLATE = "ant/{payload_type}" as const
export const QR_DOCUMENT_TEMPLATE = "ant:doc:{doc_id}:{doc_digest}" as const
export const QR_CARRIER_TEMPLATE = "ant:carrier:{carrier_type}:{value}" as const
export const CHAIN_FORMAT_VERSION = 1 as const
export const HASH_ALGORITHM = "streebog256" as const
export const DIGEST_PREFIX = "streebog256:" as const
export const SCHEMA_BASE_URI = "https://ant.invalid/contracts/" as const
export const PROBLEM_TYPE_PREFIX = "urn:ant:problem:" as const
export const TIMESTAMP_PATTERN = "^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\\.[0-9]{3}Z$" as const
export const MAX_SAFE_INTEGER = 9007199254740991 as const
