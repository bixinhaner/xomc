<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<%-- 窗口-日志文件查看 --%>
<div id="winViewPeriodCollectLogFile" style="width: 100%;height: 100%">
	 <div class="easyui-layout" data-options="border:false, fit:true">
	 	<div region="west" data-options="border:false,width:260,split:true,collapsible:false,maxWidth:500,minWidth:150">
	        <div class="easyui-panel" data-options="border:true,fit:true" style="padding: 0 15px;">
				<table id="periodCollectLogFileList"></table>
			</div>
	    </div>
	 	<div region="center" data-options="border:false" style="padding: 10px">
	 		<textarea id="periodCollectLogFileContent" class="border border-box" style="padding-left:10px;border-style:solid;width: 99%;height: 99%;resize: none;"></textarea>
	 	</div>
	 </div>
</div>