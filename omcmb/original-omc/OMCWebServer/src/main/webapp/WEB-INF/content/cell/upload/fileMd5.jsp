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
        <div region="south" data-options="border:false,height:47" style="padding: 10px">
		    <a id="cancelBtn_md5" class="easyui-linkbutton" href="javascript:void(0)" style="float:right;margin-right: 10px;"><%=rb.getString("QuXiao")%></a>
		    <a id="okBtn_md5" class="easyui-linkbutton" href="javascript:void(0)" style="float:right; margin-right: 15px;"><%=rb.getString("QueDing")%></a>
        </div>
    </div>
</div>
<script>
	$(function(){
		
	})
</script>