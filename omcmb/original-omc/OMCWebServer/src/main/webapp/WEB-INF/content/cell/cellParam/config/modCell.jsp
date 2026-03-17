<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ include file="/common/alltaglibs.jsp" %>
<ul id="paramNodesUl" class="paramNodesUl">
    <li>
        <label for="LTE_CELL_ECI_name">${LTE_CELL_ECI_name }</label>
        <input id="LTE_CELL_ECI_name" name="LTE_CELL_ECI" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_CELL_ECI_title }" 
            min_value="0" max_value="268435455" class="border border-box"/>
        <div id="LTE_CELL_ECI_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_CELL_ECI_title }
        </div>
    </li>
    <li>
        <label for="LTE_HOME_NODEB_NAME_name">${LTE_HOME_NODEB_NAME_name }</label>
        <input id="LTE_HOME_NODEB_NAME_name" name="LTE_HOME_NODEB_NAME" type="text" onblur="modcell_validateHostName(event);createMML();" title="${LTE_HOME_NODEB_NAME_title }" 
            min_length="0" max_length="48" class="border border-box"/>
        <div id="LTE_HOME_NODEB_NAME_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_HOME_NODEB_NAME_title }
        </div>
    </li>
    <li>
        <label for="LTE_TAC_name">${LTE_TAC_name }</label>
        <input id="LTE_TAC_name" name="LTE_TAC" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_TAC_title }" 
            min_value="0" max_value=65535 class="border border-box"/>
        <div id="LTE_TAC_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_TAC_title }
        </div>
    </li>
    <li>
        <label for="LTE_PHY_CELLID_LIST_name">${LTE_PHY_CELLID_LIST_name }</label>
        <input id="LTE_PHY_CELLID_LIST_name" name="LTE_PHY_CELLID_LIST" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_PHY_CELLID_LIST_title }" 
            min_value="0" max_value=503 class="border border-box"/>
        <div id="LTE_PHY_CELLID_LIST_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_PHY_CELLID_LIST_title }
        </div>
    </li>
    
	<c:if test='${hardwareVersion != "BAIBLX1.0"}'>
        <li>
            <label for="LTE_BANDS_SUPPORTED_name">${LTE_BANDS_SUPPORTED_name }</label>
            <input id="LTE_BANDS_SUPPORTED_name" name="LTE_BANDS_SUPPORTED" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_BANDS_SUPPORTED_title }" 
                min_value="1" max_value=62 class="border border-box"/>
            <div id="LTE_BANDS_SUPPORTED_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
                ${LTE_BANDS_SUPPORTED_title }
            </div>
        </li>
    </c:if>

    <li>
        <label for="LTE_FREQ_BAND_INDICATOR_name">${LTE_FREQ_BAND_INDICATOR_name }</label>
        <input id="LTE_FREQ_BAND_INDICATOR_name" name="LTE_FREQ_BAND_INDICATOR" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_FREQ_BAND_INDICATOR_title }" 
            min_value="1" max_value=62 class="border border-box"/>
        <div id="LTE_FREQ_BAND_INDICATOR_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_FREQ_BAND_INDICATOR_title }
        </div>
    </li>
    <li>
        <label for="LTE_DL_EARFCN_name">${LTE_DL_EARFCN_name }</label>
        <input id="LTE_DL_EARFCN_name" name="LTE_DL_EARFCN" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_DL_EARFCN_title }" 
            min_value="0" max_value=65535 class="border border-box"/>
        <div id="LTE_DL_EARFCN_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_DL_EARFCN_title }
        </div>
    </li>
    <li>
        <label for="LTE_UL_EARFCN_name">${LTE_UL_EARFCN_name }</label>
        <input id="LTE_UL_EARFCN_name" name="LTE_UL_EARFCN" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_UL_EARFCN_title }" 
            min_value="0" max_value=65535 class="border border-box"/>
        <div id="LTE_UL_EARFCN_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_UL_EARFCN_title }
        </div>
    </li>
    <li>
        <label for="LTE_DL_BANDWIDTH_name">${LTE_DL_BANDWIDTH_name }</label>
        <select id="LTE_DL_BANDWIDTH_name" name="LTE_DL_BANDWIDTH" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <!-- <option value="n25">5MHz</option>
            <option value="n50">10MHz</option>
            <option value="n75">15MHz</option>
            <option value="n100">20MHz</option> -->
            <option value="n25">CELL_BW_N25(5M)</option>
            <option value="n50">CELL_BW_N50(10M)</option>
            <option value="n75">CELL_BW_N75(15M)</option>
            <option value="n100">CELL_BW_N100(20M)</option>
        </select>
    </li>
    <li>
        <label for="LTE_UL_BANDWIDTH_name">${LTE_UL_BANDWIDTH_name }</label>
        <select id="LTE_UL_BANDWIDTH_name" name="LTE_UL_BANDWIDTH" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <!-- <option value="n25">5MHz</option>
            <option value="n50">10MHz</option>
            <option value="n75">15MHz</option>
            <option value="n100">20MHz</option> -->
            <option value="n25">CELL_BW_N25(5M)</option>
            <option value="n50">CELL_BW_N50(10M)</option>
            <option value="n75">CELL_BW_N75(15M)</option>
            <option value="n100">CELL_BW_N100(20M)</option>
        </select>
    </li>
    <li>
        <label for="LTE_TDD_SUBFRAME_ASSIGNMENT_name">${LTE_TDD_SUBFRAME_ASSIGNMENT_name }</label>
        <select id="LTE_TDD_SUBFRAME_ASSIGNMENT_name" name="LTE_TDD_SUBFRAME_ASSIGNMENT" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="0">0(DL:UL = 1:3)</option>
            <option value="1">1(DL:UL = 2:2)</option>
            <option value="2">2(DL:UL = 3:1)</option>
            
        </select>
    </li>
    <li>
        <label for="LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_name">${LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_name }</label>
        <select id="LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS_name" name="LTE_TDD_SPECIAL_SUB_FRAME_PATTERNS" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="5">5</option>
            <option value="7">7</option>
            
        </select>
    </li>
    <li>
        <label for="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name">${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name }</label>
        <input id="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name" name="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_title }" 
            min_value="0" max_value=837 class="border border-box"/>
        <div id="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_title }
        </div>
    </li>
<%--     <li>
        <label for="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name">${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name}</label>
        <input id="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name" name="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST" title="${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_title }" class="border border-box" 
            min_value="0" max_value="837" onblur="validateMaxAndMinValSplit(event);createMML();"/>
        <div id="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name_err" class="errSpan" style="display: none;margin-top: 5px;margin-left: 200px;">
            ${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_title}
        </div>
    </li>  --%>
    
	<c:if test='${hardwareVersion != "BAIBLX1.0"}'>
        <li>
            <label for="LTE_ENB_TYPE_name">${LTE_ENB_TYPE_name}</label>
            <select id="LTE_ENB_TYPE_name" name="LTE_ENB_TYPE" class="border border-box" onblur="createMML();">
                <option value=""></option>
                <option value="1">Home</option>
                <option value="0">Macro</option>
            </select>
        </li>
        <li>
            <label for="LTE_S1_MODE_name">${LTE_S1_MODE_name}</label>
            <select id="LTE_S1_MODE_name" name="LTE_S1_MODE" class="border border-box" >
                <option value=""></option>
                <option value="All">All</option>
                <option value="One">One</option>
            </select>
        </li>
    </c:if>
    
    <!-- qb 双工模式、功率等级、经纬度 -->
    <li class="hideItems">
        <label for="LTE_DUPLEX_MODE_name">${LTE_DUPLEX_MODE_name}</label>
        <select id="LTE_DUPLEX_MODE_name" name="LTE_DUPLEX_MODE" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="TDDMode">TDDMode</option>
            <option value="FDDMode">FDDMode</option>
        </select>
    </li>
    
    <li class="hideItems">
        <label for="LTE_CELL_POWER_MODIFY_name">${LTE_CELL_POWER_MODIFY_name}</label>
        <input id="LTE_CELL_POWER_MODIFY_name" name="LTE_CELL_POWER_MODIFY" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_CELL_POWER_MODIFY_title }" 
            min_value="17" max_value=40 class="border border-box"/>
        <div id="LTE_CELL_POWER_MODIFY_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_CELL_POWER_MODIFY_title }
        </div>
    </li>
    
    <li class="hideItems">
        <label for="LTE_CELL_LATITUDE_name">${LTE_CELL_LATITUDE_name }</label>
        <input id="LTE_CELL_LATITUDE_name" name="LTE_CELL_LATITUDE" type="text" onblur="createMML();" title="${LTE_CELL_LATITUDE_title }" 
             class="border border-box"/>
        <div id="LTE_CELL_LATITUDE_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_CELL_LATITUDE_title }
        </div>
    </li>
    
    <li class="hideItems">
        <label for="LTE_CELL_LONGITUDE_name">${LTE_CELL_LONGITUDE_name }</label>
        <input id="LTE_CELL_LONGITUDE_name" name="LTE_CELL_LONGITUDE" type="text" createMML();" title="${LTE_CELL_LONGITUDE_title }" 
             class="border border-box"/>
        <div id="LTE_CELL_LONGITUDE_name_err" class="errSpan" style="margin-top: 5px;margin-left: 200px;">
            ${LTE_CELL_LONGITUDE_title }
        </div>
    </li>
    <%-- <li class="hideItems">
        <label for="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name">${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name }</label>
        <input id="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name" name="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST" type="text" onblur="validateMaxAndMinVal(event);createMML();" title="${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_title }" 
            min_value="0" max_value=837 class="border border-box"/>
        <div id="LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_name_err" class="errSpan" style="display: none;margin-top: 5px;margin-left: 200px;">
            ${LTE_SON_PRACH_ROOT_SEQUENCE_INDEX_LIST_title }
        </div>
    </li> --%>
</ul>

<script>
    var flag = '${LTE_DUPLEX_MODE_name}';
    if(flag){
        $(".hideItems").show();
    }else{
        $(".hideItems").hide();
    }
    //如果SAS开关打开，则该参数不可配置
    if(SASEnble == "1"){
        $("#LTE_DL_EARFCN_name,#LTE_UL_EARFCN_name,#LTE_FREQ_BAND_INDICATOR_name ,#LTE_BANDS_SUPPORTED_name").attr("disabled",true);
        $("#LTE_DL_BANDWIDTH_name,#LTE_UL_BANDWIDTH_name").attr("disabled",true).css("background","#EAF1F4");
    }
    function modcell_validateHostName(e){
        var ele = $(e["target"]);
        var currVal = ele.val();
        var reg = /^[A-Za-z]{1}[A-Za-z0-9_\-]{0,47}$/;
        if(currVal!=""){
            if(!reg.test(currVal)){
                 $("#" + ele.attr("id") + "_err").show();
                 ele.addClass("err_border");
            }else{
                 $("#" + ele.attr("id") + "_err").hide();
                 ele.removeClass("err_border");
            }
        }else{
             ele.removeClass("err_border");
        }
    }
</script>