<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%-- 导入文件进行参数配置 --%>
<script type="text/javascript">
    var ctx = "${ctx}";
    var isEnglish = <%=(rb.getLocale().equals(Locale.ENGLISH))%>;
    var QingXuanZeSheBei = "<%=rb.getString("QingXuanZeSheBei")%>";
    var TiShi = "<%=rb.getString("TiShi")%>";
    var CaoZuoKeNengYinQiZiDongChongQi = "<%=rb.getString("CaoZuoKeNengYinQiZiDongChongQi")%>";
</script>
<script type="text/javascript" src="${ctx}/js/bi/cell/cellParamFileConfig.js"></script>
<%-- 用于上传文件的Form表单 --%>
<form enctype="multipart/form-data" method="post" id="uploadForm"
          action="${ctx}/cell/param/uploadParamFile.action" style="display: none;">
    <input id="uploadFile" name="uploadFile" type="file">
</form>
<%-- 整体布局 --%>
<div class="easyui-panel" data-options="fit:true,border:false">
	<div class="easyui-layout" data-options="fit:true,border:false">
	    <div region="north" data-options="border: true, height: 35" style="border-width: 0 0 1px 0;line-height:32px;overflow: hidden;padding:0 10px;">
	    	<label for="txt_paramFilePath"><%=rb.getString("ShangChuanCLIWenJian")%></label>
	    	<input id="txt_paramFilePath" class="border-box border" readonly="readonly" type="text" style="vertical-align: middle;height:24px;"
	    			onclick="scanClick()"/>
	    </div>
	    <div region="center" data-options="border: false">
            <table class="easyui-datagrid" id="tableCmdLineResult" fit="true" data-options="border:false,fitColumns:true,
	                    rownumbers:true,
	                    singleSelect:true,
	                    striped:true,onLoadSuccess:datagridLoadSuccess">
                <thead>
                <tr>
                    <th data-options="field:'lineNo'" width="50"><%=rb.getString("HangHao")%></th>
                    <th data-options="field:'line'" width="200"><%=rb.getString("MingLing")%></th>
                    <th data-options="field:'success',formatter:successFormatter,styler:successStyler" width="50"><%=rb.getString("ChengGong")%></th>
                    <th data-options="field:'errInfo'" width="200"><%=rb.getString("ShiBaiXinXi")%></th>
                </tr>
                </thead>
            </table>
        </div>
	</div>
</div>