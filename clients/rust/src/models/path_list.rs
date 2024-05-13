/*
 * lakeFS API
 *
 * lakeFS HTTP API
 *
 * The version of the OpenAPI document: 1.0.0
 * Contact: services@treeverse.io
 * Generated by: https://openapi-generator.tech
 */

use crate::models;

#[derive(Clone, Default, Debug, PartialEq, Serialize, Deserialize)]
pub struct PathList {
    #[serde(rename = "paths")]
    pub paths: Vec<String>,
}

impl PathList {
    pub fn new(paths: Vec<String>) -> PathList {
        PathList {
            paths,
        }
    }
}
