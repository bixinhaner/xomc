<%@ page language="java" contentType="text/html; charset=UTF-8"
    pageEncoding="UTF-8"%>
<%@ taglib uri="http://java.sun.com/jsp/jstl/core" prefix="c"%>
<ul id="paramNodesUl" class="paramNodesUl">
    <%-- 小区选择UE最大功率 --%>
    <li>
        <label for="LTE_PMAX_name">${LTE_PMAX_name }</label>
        <input id="LTE_PMAX_name" name="LTE_PMAX" title="${LTE_PMAX_title }" class="border border-box" 
            min_value="-30" max_value="33" onblur="validateMaxAndMinVal(event);createMML();"/>
        <div id="LTE_PMAX_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_PMAX_title }
        </div>
    </li>
    
    <!-- 参考信号功率 -->
    <%-- <li>
        <label for="LTE_REFERENCE_SIG_POWER_name">${LTE_REFERENCE_SIG_POWER_name }</label>
        <input id="LTE_REFERENCE_SIG_POWER_name" name="LTE_REFERENCE_SIG_POWER" title="${LTE_REFERENCE_SIG_POWER_title }" class="border border-box" 
            min_value="-60" max_value="50" onblur="validateMaxAndMinVal(event);createMML();"/>
        <div id="LTE_REFERENCE_SIG_POWER_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_REFERENCE_SIG_POWER_title }
        </div>
    </li> --%>
    
    <%-- PRACH功率攀升的步长 --%>
    <li>
        <label for="LTE_POWER_RAMPING_name">${LTE_POWER_RAMPING_name }</label>
        <select id="LTE_POWER_RAMPING_name" name="LTE_POWER_RAMPING" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="0">0</option>
            <option value="2">2</option>
            <option value="4">4</option>
            <option value="6">6</option>
        </select>
    </li>
    <%-- 初始接收目标功率 --%>
    <li>
        <label for="LTE_PREAMBLE_INIT_TARGET_POWER_name">${LTE_PREAMBLE_INIT_TARGET_POWER_name }</label>
        <select id="LTE_PREAMBLE_INIT_TARGET_POWER_name" name="LTE_PREAMBLE_INIT_TARGET_POWER" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="-120">dBm-120</option>
            <option value="-118">dBm-118</option>
            <option value="-116">dBm-116</option>
            <option value="-114">dBm-114</option>
            <option value="-112">dBm-112</option>
            <option value="-110">dBm-110</option>
            <option value="-108">dBm-108</option>
            <option value="-106">dBm-106</option>
            <option value="-104">dBm-104</option>
            <option value="-102">dBm-102</option>
            <option value="-100">dBm-100</option>
            <option value="-98">dBm-98</option>
            <option value="-96">dBm-96</option>
            <option value="-94">dBm-94</option>
            <option value="-92">dBm-92</option>
            <option value="-90">dBm-90</option>
        </select>
    </li>
    <%-- PUSCH标称功率P0 --%>
    <li>
        <label for="LTE_PO_NOMINAL_PUSCH_name">${LTE_PO_NOMINAL_PUSCH_name }</label>
        <input id="LTE_PO_NOMINAL_PUSCH_name" name="LTE_PO_NOMINAL_PUSCH" title="${LTE_PO_NOMINAL_PUSCH_title }" class="border border-box" 
            min_value="-126" max_value="24" onblur="validateMaxAndMinVal(event);createMML();"/>
        <div id="LTE_PO_NOMINAL_PUSCH_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_PO_NOMINAL_PUSCH_title }
        </div>
    </li>
    <%-- PICCH标称功率P0 --%>
    <li>
        <label for="LTE_PO_NOMINAL_PUCCH_name">${LTE_PO_NOMINAL_PUCCH_name }</label>
        <input id="LTE_PO_NOMINAL_PUCCH_name" name="LTE_PO_NOMINAL_PUCCH" title="${LTE_PO_NOMINAL_PUCCH_title }" class="border border-box" 
            min_value="-127" max_value="-96" onblur="validateMaxAndMinVal(event);createMML();"/>
        <div id="LTE_PO_NOMINAL_PUCCH_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_PO_NOMINAL_PUCCH_title }
        </div>
    </li>
    <%-- 路损补偿因子 --%>
    <li>
        <label for="LTE_ALPHA_name">${LTE_ALPHA_name }</label>
        <select id="LTE_ALPHA_name" name="LTE_ALPHA" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="0">0</option>
            <option value="40">40</option>
            <option value="50">50</option>
            <option value="60">60</option>
            <option value="70">70</option>
            <option value="80">80</option>
            <option value="90">90</option>
            <option value="100">100</option>
        </select>
    </li>
    <%-- 最大路损 --%>
    <li>
        <label for="LTE_MAX_PATHLOSS_name">${LTE_MAX_PATHLOSS_name }</label>
        <input id="LTE_MAX_PATHLOSS_name" name="LTE_MAX_PATHLOSS" title="${LTE_MAX_PATHLOSS_title }" class="border border-box" 
            min_value="100" max_value="135" onblur="validateMaxAndMinVal(event);createMML();"/>
        <div id="LTE_MAX_PATHLOSS_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_MAX_PATHLOSS_title }
        </div>
    </li>
    <%-- 最大路损下的基础目标信噪比 --%>
    <li>
        <label for="LTE_TARGET_UL_SINR_name">${LTE_TARGET_UL_SINR_name }</label>
        <input id="LTE_TARGET_UL_SINR_name" name="LTE_TARGET_UL_SINR" title="${LTE_TARGET_UL_SINR_title }" class="border border-box" 
            min_value="-6" max_value="10" onblur="validateMaxAndMinVal(event);createMML();"/>
        <div id="LTE_TARGET_UL_SINR_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_TARGET_UL_SINR_title }
        </div>
    </li>
    <%-- Po_ue_pucch --%>
    <li>
        <label for="LTE_PO_UE_PUCCH_name">${LTE_PO_UE_PUCCH_name }</label>
        <input id="LTE_PO_UE_PUCCH_name" name="LTE_PO_UE_PUCCH" title="${LTE_PO_UE_PUCCH_title }" class="border border-box" 
            min_value="-8" max_value="7" onblur="validateMaxAndMinVal(event);createMML();"/>
        <div id="LTE_PO_UE_PUCCH_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_PO_UE_PUCCH_title }
        </div>
    </li>
    <%-- Po_ue_pusch --%>
    <li>
        <label for="LTE_PO_UE_PUSCH_name">${LTE_PO_UE_PUSCH_name }</label>
        <input id="LTE_PO_UE_PUSCH_name" name="LTE_PO_UE_PUSCH" title="${LTE_PO_UE_PUSCH_title }" class="border border-box" 
            min_value="-8" max_value="7" onblur="validateMaxAndMinVal(event);createMML();"/>
        <div id="LTE_PO_UE_PUSCH_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_PO_UE_PUSCH_title }
        </div>
    </li>
    <%-- 无线端口信号功率比 --%>
    <li>
        <label for="LTE_PB_name">${LTE_PB_name }</label>
        <select id="LTE_PB_name" name="LTE_PB" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="0">0</option>
            <option value="1">1</option>
            <option value="2">2</option>
            <option value="3">3</option>
        </select>
    </li>
    <%-- 小区PDSCH采用固定功率分配时的PA取值 --%>
    <li>
        <label for="LTE_PA_name">${LTE_PA_name }</label>
        <select id="LTE_PA_name" name="LTE_PA" class="border border-box" onblur="createMML();">
            <option value=""></option>
            <option value="-600">-600</option>
            <option value="-477">-477</option>
            <option value="-300">-300</option>
            <option value="-177">-177</option>
            <option value="0">0</option>
            <option value="100">100</option>
            <option value="200">200</option>
            <option value="300">300</option>
        </select>
    </li>
    <%-- 功率等级 --%>
    <c:if test='${hardwareVersion == "EA4.0" }'>
	    <li>
	        <label for="LTE_CELL_POWER_MODIFY_name">${LTE_CELL_POWER_MODIFY_name }</label>
	        <input id="LTE_CELL_POWER_MODIFY_name" name="LTE_CELL_POWER_MODIFY" title="${LTE_CELL_POWER_MODIFY_title }" class="border border-box" 
	            min_value="0" max_value="40" onblur="validateMaxAndMinVal(event);createMML();"/>
	        <div id="LTE_CELL_POWER_MODIFY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
	            ${LTE_CELL_POWER_MODIFY_title }
	        </div>
	    </li>
    </c:if>
    <c:if test='${hardwareVersion == "EA4.0DUAL" }'>
	    <li>
	        <label for="LTE_CELL_POWER_MODIFY_name">${LTE_CELL_POWER_MODIFY_name }</label>
	        <input id="LTE_CELL_POWER_MODIFY_name" name="LTE_CELL_POWER_MODIFY" title="${LTE_CELL_POWER_MODIFY_title }" class="border border-box" 
	            min_value="0" max_value="43" onblur="validateMaxAndMinVal(event);createMML();"/>
	        <div id="LTE_CELL_POWER_MODIFY_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
	            ${LTE_CELL_POWER_MODIFY_title }
	        </div>
	    </li>
    </c:if>
    <%-- 上行接收增益 --%>
    <li>
        <label for="LTE_PHY_RXGAIN_name">${LTE_PHY_RXGAIN_name }</label>
        <input id="LTE_PHY_RXGAIN_name" name="LTE_PHY_RXGAIN" title="${LTE_PHY_RXGAIN_title }" class="border border-box" 
            min_value="-48" max_value="48" onblur="validateMaxAndMinVal(event);createMML();"/>
        <div id="LTE_PHY_RXGAIN_name_err" class="errSpan" style="display: block;margin-top: 5px;margin-left: 200px;">
            ${LTE_PHY_RXGAIN_title }
        </div>
    </li>
</ul>
