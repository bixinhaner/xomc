<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp"%>


<div class="easyui-panel" data-options="border:false,fit:true" style="text-align: center;padding: 20px">
    <div class="about_logo"></div>
	<div style="margin:20px 0 10px 0;">
		<span><%=rb.getString("BanBen")%> <%=rb.getString("MaoHao")%></span><span class="about_version"></span>
	</div>
	<div>
		<span><%=rb.getString("MingCheng")%> <%=rb.getString("MaoHao")%></span><span class="about_name"></span>
	</div>
</div>

<script>
	var uiType ="<%= currOMCLicInfo.uiType%>";
	var logoClass = {
			"0" : "nologo_about",
			"1" : "baicells_about",
			"2" : "rongyu_about",
			"3" : "tianyi_about",
			"4" : "rihai_about",
			"5" : "fenghuo_about"
	};
	
	var versionText = {
			"0" : "OMC_",
			"1" : "BaiOMC_",
			"2" : "OMC_",
			"3" : "TYOMC-",
			"4" : "sunsea aiot ",
			"5" : "OMC_"
	};
	
	var nameText = {
			"0" : "OMC",
			"1" : "BaiOMC",
			"2" : "OMC",
			"3" : "OMC",
			"4" : "sunsea aiot",
			"5" : "OMC"
	}
	
	$(".about_logo").addClass(logoClass[uiType]);
	$(".about_version").text(versionText[uiType] + '${omc_ver}');
	$(".about_name").text(nameText[uiType]);

</script>