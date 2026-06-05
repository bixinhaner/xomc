/**
 * xmlRootAttr — 从 XML 文件头部提取根元素属性(浏览器端轻量预检)。
 *
 * 导入 XML 调整(2026-06-05):三库上传取消手填「名称」,名称唯一来源是 XML 根元素属性
 * (告警 neType / KPI platform / 参数模型 paramModel)。前端选文件后用本工具提取属性,
 * 用于:① 展示"将保存为 <属性值>.xml" ② 本地查重预检(后端 409 仍是真值源)。
 *
 * 只读文件头 4 KB(根元素及其属性总在文件头),正则容错:提取失败返 null,
 * 由调用方回退"交给后端校验"。
 */

const HEAD_BYTES = 4096;

/**
 * 提取 XML 根元素上的指定属性值;读不到 / 属性缺失返 null。
 * @param file 用户选择的 XML File
 * @param attr 属性名(neType / platform / paramModel / deviceType ...)
 */
export async function extractXmlRootAttr(file: File, attr: string): Promise<string | null> {
  try {
    const head = await file.slice(0, HEAD_BYTES).text();
    // 跳过 <?xml ?> 声明与注释,取第一个元素标签内的属性
    const re = new RegExp(`<[A-Za-z][^>]*?\\b${attr}\\s*=\\s*"([^"]*)"`, 'i');
    const m = head.match(re);
    const v = m?.[1]?.trim();
    return v ? v : null;
  } catch {
    return null;
  }
}
