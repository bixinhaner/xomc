<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<%--收集 -选择基站窗口 --%>
<div id="wineNodBSelection" style="width: 100%;height: 100%">
	<div class="easyui-layout" data-options="fit:true">
        <div region="center" data-options="border:false" style="padding:10px 20px;">
            <table id="gridCell_collection"></table>
        </div>
        <div region="south" data-options="border:false,height:40" style="padding:0 20px;">
            <a onclick="stepDown_logSelection()" style="float:right;" class="easyui-linkbutton"><%=rb.getString("XiaYiBu")%></a>
        </div>
    </div>
</div>