<%@page import="com.baicells.omc.SpringCtxHolder"%>
<%@page import="com.baicells.omc.busi.system.au.license.util.LicenseCons"%>
<%@page import="com.baicells.omc.busi.system.au.CurrOMCLicInfo"%>
<%@page import="com.baicells.omc.busi.utils.ApplicationProperties"%>
<%@page import="com.baicells.omc.framework.utils.CommMethod"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/fmt" prefix="fmt"%>
<%@page import="com.baicells.omc.busi.system.utils.UserUtil"%>
<%@page import="java.util.ResourceBundle"%>
<%@page import="java.util.Locale"%>
<c:set var="ctx" value="${pageContext.request.contextPath}" />
<c:set var="pageSize" value="100" />
<c:set var="pageList" value="[50,100,200,500]" />
<c:set var="omc_ver" value="12.2.5" />
<%
	CurrOMCLicInfo currOMCLicInfo = SpringCtxHolder.getBean(CurrOMCLicInfo.class);
    // Load Internationalization Resource package
    String manufacturer = currOMCLicInfo.getUiType();
	ApplicationProperties prop = SpringCtxHolder.getBean(ApplicationProperties.class);
	String vc = prop.getValidatecode();
    String validateCode = CommMethod.isEmpty(vc) ? "0" : CommMethod.nvl(vc);
	String filterSpecialCharactersEnable = currOMCLicInfo.filterSpecialCharactersEnable;
	
    HttpSession sion = request.getSession();
    Locale locale = null;
    if (sion.getAttribute("language_code") != null && !"".equals(sion.getAttribute("language_code"))) {
        locale = new Locale(sion.getAttribute("language_code").toString());
    } else {
        locale = request.getLocale();
    }
	
   // ResourceBundle rb = ResourceBundle.getBundle("i18n.resource." + manufacturer + ".display_language", locale);
 	ResourceBundle rb = ResourceBundle.getBundle("messages", locale);
 
    String language = locale.toString();
    if (language.contains("zh")) {
%>


<c:set var="i18n_type" value="zh" />
<%
    } else {
%>
<c:set var="i18n_type" value="en" />
<%
    }
%>

<%@ include file="/common/variables.jsp"%>
<%
	String subScriptPage = "/skin/" + manufacturer + "/skin_script.jsp";
%>


<c:set var="validatecode" value="<%=validateCode%>" />
<script type="text/javascript">
var filterSpecialCharactersEnable = '${filterSpecialCharactersEnable}';
//回车表单是否自动提交
function isAutoSubmit() {
    var userAgent = navigator.userAgent; //取得浏览器的userAgent字符�?
    var isIE = userAgent.indexOf("compatible") > -1 && userAgent.indexOf("MSIE") > -1; //判断是否IE<11浏览�?
    var isEdge = userAgent.indexOf("Edge") > -1 && !isIE; //判断是否IE的Edge浏览�?
    var isIE11 = userAgent.indexOf('Trident') > -1 && userAgent.indexOf("rv:11.0") > -1;
    if(isIE) {
        return false;
    } else if(isEdge) {
        return true
    } else if(isIE11) {
        return true
    }else{
        return true
    }
}
</script>


