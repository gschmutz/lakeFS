package io.treeverse.gc

import org.apache.hadoop.fs.Path
import org.apache.spark.sql.DataFrame
import org.apache.spark.sql.SparkSession
import org.apache.spark.sql.functions._

import scala.collection.mutable.ListBuffer
import io.treeverse.clients.ConfigMapper
import org.apache.hadoop.fs.FileStatus

/** List all the files under a given path.
 */
abstract class DataLister {
  @transient lazy val spark: SparkSession = SparkSession.active
  def listData(configMapper: ConfigMapper, path: Path): DataFrame
}

class NaiveDataLister extends DataLister {
  override def listData(configMapper: ConfigMapper, path: Path): DataFrame = {
    import spark.implicits._
    val fs = path.getFileSystem(configMapper.configuration)
    val dataIt = fs.listFiles(path, false)
    val dataList = new ListBuffer[(String, Long)]()
    while (dataIt.hasNext) {
      val fileStatus = dataIt.next()
      dataList += ((fileStatus.getPath.getName, fileStatus.getModificationTime))
    }
    dataList.toDF("address", "last_modified")
  }
}

class FileDescriptor(val path: String, val lastModified: Long) extends Serializable

class ParallelDataLister extends DataLister with Serializable {
  def listPath(
      configMapper: ConfigMapper,
      p: Path
  ): List[FileDescriptor] = {
    val fs = p.getFileSystem(configMapper.configuration)
    if (!fs.exists(p)) {
      return List[FileDescriptor]()
    }
    fs.listStatus(p)
      .map((status: FileStatus) =>
        new FileDescriptor(status.getPath.getName, status.getModificationTime)
      )
      .toList
  }

  override def listData(configMapper: ConfigMapper, path: Path): DataFrame = {
    import spark.implicits._
    val slices = listPath(configMapper, path)
    val pathStr = path.toString
    val objectsUDF = udf((sliceId: String) => {
      val slicePath = new Path(pathStr, sliceId)
      listPath(configMapper, slicePath)
        .map(s => (s.path, s.lastModified))
    })
    val objectsDF = slices
      .map(_.path)
      .toDF("slice_id")
      .withColumn("udf", explode(objectsUDF(col("slice_id"))))
      .withColumn("base_address", concat(col("slice_id"), lit("/"), col("udf._1")))
      .withColumn("last_modified", col("udf._2"))
      .select("base_address", "last_modified")
    objectsDF
  }
}