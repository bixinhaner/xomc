<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<%-- 窗口-上传配置参数文件，导入后的结果显示 --%>
<div id="winImportSelfstartResult" style="width: 100%;height: 100%;">
	<div class="easyui-layout" data-options="fit:true,border:false">
		<div region="north" data-options="border:false,height:50" style="padding: 5px 20px;">
			<div id="suc_count_importSelfstartResult"></div>
			<div id="unsuc_count_importSelfstartResult"></div>
		</div>
		<div region="center" data-options="border:false" style="padding: 0px 20px;">
			<div id="unsucSelfstart_info"></div>
		</div>
		<div region="south" data-options="border:false,height:10">
		</div>
	</div>
</div>