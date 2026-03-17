<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="/common/alltaglibs.jsp" %>
<ul id="paramNodesUl" class="paramNodesUl">
	<li>
		<label for="LTE_SELFSTART_ENABLE_name">${LTE_SELFSTART_ENABLE_name }</label>
		<select id="LTE_SELFSTART_ENABLE_name" name="LTE_SELFSTART_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="LTE_SELFCONFIG_PHY_CELLID_ENABLE_name">${LTE_SELFCONFIG_PHY_CELLID_ENABLE_name }</label>
		<select id="LTE_SELFCONFIG_PHY_CELLID_ENABLE_name" name="LTE_SELFCONFIG_PHY_CELLID_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li>
		<label for="LTE_SELFCONFIG_PRACH_ENABLE_name">${LTE_SELFCONFIG_PRACH_ENABLE_name }</label>
		<select id="LTE_SELFCONFIG_PRACH_ENABLE_name" name="LTE_SELFCONFIG_PRACH_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	<li class="hideItems">
		<label for="LTE_SON_PHY_CELLID_LIST_name">${LTE_SON_PHY_CELLID_LIST_name }</label>
		<input id="LTE_SON_PHY_CELLID_LIST_name" name="LTE_UL_EARFCN" type="text" onblur="createMML();" title="${LTE_SON_PHY_CELLID_LIST_title }" 
			 class="border border-box"/>
		<div id="LTE_SON_PHY_CELLID_LIST_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
			${LTE_SON_PHY_CELLID_LIST_title }
		</div>
	</li>
	<li>
		<label for="LTE_SELFCONFIG_EARFCN_ENABLE_name">${LTE_SELFCONFIG_EARFCN_ENABLE_name }</label>
		<select id="LTE_SELFCONFIG_EARFCN_ENABLE_name" name="LTE_SELFCONFIG_EARFCN_ENABLE" class="border border-box" onblur="createMML();">
			<option value=""></option>
			<option value="1">true</option>
			<option value="0">false</option>
		</select>
	</li>
	
</ul>


<script>
	var flag = '${LTE_SON_PHY_CELLID_LIST_name }';
	if(flag){
		$(".hideItems").show();
	}else{
		$(".hideItems").hide();
	}
</script>