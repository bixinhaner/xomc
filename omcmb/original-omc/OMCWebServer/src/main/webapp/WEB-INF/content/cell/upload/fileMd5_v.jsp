<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<div id="inputFileMd5" style="width: 100%;height: 100%;">
    <div class="easyui-layout" data-options="fit:true,border:false">
        <div region="center" data-options="border:false" style="padding: 10px 20px;line-height: 30px;">
            <div style="height:28px;line-height:28px;margin:20px auto;text-align:center;">
		        <span><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%></span>
		        <div id="MD5Value" style="display:inline-block;margin-left:5px"></div>
	        </div>
        </div>
        <div region="south" data-options="border:true,height:38" style="border-width: 1px 0 0 0; padding: 5px 10px;">
		    <a class="easyui-linkbutton" href="javascript:void(0)" onclick="cancelInputFile()" style="float:right; margin-right: 5px;"><%=rb.getString("QuXiao")%></a>
		    <a class="easyui-linkbutton" href="javascript:void(0)" onclick="saveInputFile()" style="float:right; margin-right: 5px;"><%=rb.getString("QueDing")%></a>
        </div>
    </div>
</div>