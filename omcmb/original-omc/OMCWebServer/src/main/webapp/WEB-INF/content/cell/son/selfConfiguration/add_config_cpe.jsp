<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
    #cpeConfigPage .split-line { border: none; border-top: 1px solid #e9e9e9; margin: 10px 0 20px 0; }
    #cpeConfigPage .el-select-mini input { min-height: 26px; }
    #cpeConfigPage .basicInfoBox { margin: 16px 40px 10px; }
	#cpeConfigPage .originalBox .el-checkbox { padding: 3px 6px 0 44px; }
	#cpeConfigPage .addVersionWarp { width: 600px; height: 254px; border-radius: 4px; text-align: center; margin-left: 150px; border: 1px solid #E9EDF9;}
	#cpeConfigPage .addVersionBtn { font-size: 20px; padding: 100px 0 10px; }
	#cpeConfigPage .selectVersionBox .el-form-item__content { margin-left: 0px !important; }
	#cpeConfigPage .selectVersionBox .el-form-item__content .el-input { width: 300px; }
	#cpeConfigPage .selectMethodBox { display: flex; flex-direction: row; justify-content: space-between; width: 70%; }
	#cpeConfigPage .selectMethodBox .el-radio-button__inner { display: flex; border: 1px solid #E9EDF9; background: #F5F7FE; min-width: 240px; height: 50px; line-height: 50px;  border-radius: 4px; padding: 0; font-size: unset; font-weight: normal;text-align: center;}
	#cpeConfigPage .selectMethodBox .el-radio-button__inner [class*=el-icon-]+span { margin-left: 0; }
	#cpeConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .methodTitle {color:var(--main-color);}
	#cpeConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .commonIconStyle,
	#cpeConfigPage .selectMethodBox .el-radio-button__inner:hover,
	#cpeConfigPage .selectMethodBox .el-radio-button__inner:hover .commonIconStyle,
	#cpeConfigPage .el-radio:hover,
	#cpeConfigPage .el-radio .el-radio__inner:hover {
		color:var(--main-color);
		background: rgba(var(--main-color-rgba1),0.1);
		border-color:var(--main-color) !important;
    }
	#cpeConfigPage .newIconBoxCls-bt  .el-icon-circle-close:before {
		content: '\e778';
		font-size: 12px !important;
	}
	#cpeConfigPage .el-icon-menu-system:before {
		color: #7A7992;
	}
	#cpeConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .el-icon-menu-system:before,
	#cpeConfigPage .selectMethodBox .el-radio-button__inner:hover .el-icon-menu-system:before {
		color:var(--main-color);
	}
	#cpeConfigPage .licenseImportBox { margin: 0 10px; width: 26px; height: 26px; border: 1px solid #D7D7E6; text-align: center; border-radius: 8px; }
	#cpeConfigPage .licenseImportBox i { font-size: 12px; line-height: 26px; }
	#cpeConfigPage .leftWarp {flex: 1; height: 100%; position: relative; overflow: hidden; flex-direction: column; border-radius: 10px; border: 1px solid #E9EDF9;background: #FFFFFF;}
	#cpeConfigPage .titleBox { width: 100%; height: 38px; background: #FFFFFF; line-height: 38px; position: absolute; top: 0; left: 0; z-index: 100; }
	#cpeConfigPage .footerBox { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
	#cpeConfigPage .AddTitle { padding: 0 30px; }
	#cpeConfigPage .closeIconBox .el-icon { font-size: 14px !important; color: #7a7992; margin: 12px 16px;}
	#cpeConfigPage .specifyModifyRightBox { flex: 0 640px; height: 100%;  margin-left: 10px; border-radius: 10px; overflow: hidden; }
	#cpeConfigPage .rightBox { flex: 0 1 360px; height: 100%;  margin-left: 10px; border-radius: 10px; overflow: hidden; border: 1px solid #E9EDF9; background: #FFFFFF;}
	#cpeConfigPage .mainContent { width: 100%; overflow-y: scroll; height: 100%; position: absolute; top: 40px; z-index: 0; box-sizing: border-box; }
	#cpeConfigPage .commonTitle { height: 38px; line-height: 38px; }
	#cpeConfigPage .rightHeaderBox .addTitle {  padding: 0; }
	#cpeConfigPage .rightTextBox { padding: 10px 0; }
	#cpeConfigPage .commonBorderBottom { border-bottom: 1px solid #E9EDF9; }
	#cpeConfigPage .commonBorderTop { border-top: 1px solid #E9EDF9; }
	#cpeConfigPage .originalVersionBox { padding: 16px 0 0; }
	#cpeConfigPage .originalVersionBox .el-input{ width: 320px; }
	#cpeConfigPage .originalVersionBox .el-input__inner{ height: 30px; line-height: 30px; }
	#cpeConfigPage .originalVersionBox .el-input-group__append { padding: 0 8px; background-color: #FFFFFF; }
	#cpeConfigPage .originalVersionBox .el-input-group__append .el-icon { font-size: 14px; }
	#cpeConfigPage .originalVersionBox .el-input-group__append .el-icon:before { color: #7A7992; }
	#cpeConfigPage .originalVersionBox .el-table tr { display: none; }
	#cpeConfigPage .ipErrorTip { color: #FA5555; font-size: 12px; }
	#cpeConfigPage .versionResultBox { border-radius: 4px; background-color: #FFFFFF; margin-top: 5px; width: 318px; max-height: 122px; padding: 5px 0; overflow: auto; border: 1px solid #E9EDF9; }
	#cpeConfigPage .versionResultBox .el-form-item { margin-right: 0 !important;}
	#cpeConfigPage .form-suffix { position: relative; margin: 3px 0 0 20px; padding: 0; width:278px; border: none; background: #fff; }
	#cpeConfigPage .form-suffix:hover { background: #F4F9FF; border-radius: 100px; }
	#cpeConfigPage .form-suffix .deleteVersion { display: none; position: absolute; right: 0; top: 0px;  color: var(--main-color); font-size: 14px;}
	#cpeConfigPage .form-suffix:hover .deleteVersion { display: inline-block !important; }
	#cpeConfigPage .form-suffix .text { padding-left: 10px; color: #666666; }
	#cpeConfigPage .selectedVersion .queryGroup .el-input__inner { width: 220px !important; }
	#cpeConfigPage .selectedVersion .el-query .advanceQuery { height: 28px; }
	#cpeConfigPage .selectedVersion .el-query .advanceQuery .el-input.el-input--small { width: 230px !important; }
	#cpeConfigPage .selectedVersion .el-query { right: 0px; }
	#cpeConfigPage .operBtn { width: 26px; height: 26px; border: 1px solid #D7D7E6; border-radius: 8px; margin-top: 1px; text-align: center; }
	#cpeConfigPage .operBtn .el-icon { line-height: 26px; }
	#cpeConfigPage .operBtn .el-icon:before,
	#cpeConfigPage .importIcon:before { color: #7A7992; }
	.container .group { padding-left: 26px; }
	.mainContent .el-form-item { margin-bottom: 22px; }
	.mainContent .el-form-item__label { line-height: 28px; } 
	#cpeConfigPage .el-radio.is-bordered { max-width: 160px; height: 30px; padding: 7px 12px; }
	#cpeConfigPage .el-radio-group .el-radio__label { font-size: 12px; }
	#cpeConfigPage .enableCommon .el-switch { margin-top: 4px; }
	#cpeConfigPage .selectCommon .el-select .el-input { width: 150px; }
	#cpeConfigPage .inputCommon .el-input__inner { border-radius: 4px; }
	#cpeConfigPage .inputCommon .el-form-item__label { width: 150px; }
    #cpeConfigPage  .paramTwoBox { padding: 10px 20px; }
    #cpeConfigPage .infoSpecifiedDevice .el-collapse-item__arrow { position: absolute; left: 0; top: 0px; }
    #cpeConfigPage .infoSpecifiedDevice .el-collapse-item {  position: relative; }
    #cpeConfigPage .infoSpecifiedDevice .el-icon-arrow-right {  font-size: 16px; }
    #cpeConfigPage .infoSpecifiedDevice .el-icon-arrow-right:before { content: "\e639"; color: #BBB;  }
    #cpeConfigPage .infoSpecifiedDevice .is-active.el-icon-arrow-right:before { content: "\e638"; color: #BBB; }
    #cpeConfigPage .infoSpecifiedDevice .el-collapse-item__arrow.is-active { transform: rotate(0deg); }
    #cpeConfigPage .infoSpecifiedDevice .el-collapse { border-top: 1px solid #fff; border-bottom: 1px solid #fff; }
    #cpeConfigPage .infoSpecifiedDevice .el-collapse-item__wrap { border-bottom: 1px solid #fff; }
    #cpeConfigPage .infoSpecifiedDevice .btn-next .el-icon-arrow-right:before { content: "\e794"; }
	#cpeConfigPage .commonText { padding: 3px 0; }
    #cpeConfigPage .commonText1 { padding: 0 10px; }
	#cpeConfigPage .el-collapse-item__content { padding-bottom: 0; }
	#cpeConfigPage .enbIdItem .el-textarea__inner { height: 150px; resize: none; border-radius: 4px; }
	#cpeConfigPage .reginDeployingBox .el-tabs--border-card { border: 1px solid #E9EDF9; box-shadow: none; -webkit-box-shadow: none; }
	#cpeConfigPage .reginDeployingBox .el-tabs .el-tabs__header,
	#cpeConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header { border-bottom: 1px solid #E9EDF9; }
	#cpeConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header { background-color: #FFFFFF; }
	#cpeConfigPage .reginDeployingBox .el-tabs__item { font-size: 14px; color: #7A7992; font-weight: normal;  }
	#cpeConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header .el-tabs__item.is-active { color: #7A7992; font-weight: bold;  border-right-color: #E9EDF9; border-left-color: #E9EDF9; }
	#cpeConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header .el-tabs__item:not(.is-disabled):hover { color: #4D84FF; }
	#cpeConfigPage .modifyForm .el-form-item { margin-bottom: 20px; display: inline-block; margin-right: 80px; }
	#cpeConfigPage .modifyForm .el-form-item .el-form-item__label { margin-bottom: 4px; }
	#cpeConfigPage .table-suffix { border: none; position: relative; }
	#cpeConfigPage .table-suffix .deleteVersion { display: none; position: absolute; right: 0; top: 2px;  color: var(--main-color); font-size: 14px;}
	#cpeConfigPage .table-suffix:hover .deleteVersion { display: inline-block !important; }
	#cpeConfigPage .table-suffix .text { padding-left: 10px; color: #666666; }
	#cpeConfigPage .disabledClass { cursor: not-allowed !important; opacity: 0.4; }
	#cpeConfigPage .disabledClass:before {color: #c0c4cc; cursor: not-allowed !important; }
	#cpeConfigPage .defaultClass { cursor: pointer; }
	#cpeConfigPage .el-checkbox-button { margin: 6px 6px 6px 0; }
	
	#cpeConfigPage .el-checkbox-button__inner { border-radius: 100px; font-size: 12px; color: #333333; padding: 7px 16px; border: 1px solid #DFE2EE; background: #F6F8FD; }
	#cpeConfigPage .el-checkbox-button.is-checked .el-checkbox-button__inner { background-color: #E4F1FF; border-radius: 100px; box-shadow: none; -webkit-box-shadow: none; color: #333333; border: 1px solid #6EB3FF;}
	#cpeConfigPage .el-checkbox-button.is-checked .el-checkbox-button__inner,
	#cpeConfigPage .el-checkbox-button .el-checkbox-button__inner:hover { color : var(--main-color); border-color:var(--main-color) !important;background: rgba(var(--main-color-rgba1),0.1);}
	#cpeConfigPage .el-checkbox-button:first-child .el-checkbox-button__inner,
	#cpeConfigPage .el-checkbox-button:last-child .el-checkbox-button__inner { border-radius: 100px; }
	#cpeConfigPage .el-checkbox-button.is-focus .el-checkbox-button__inner { border: 1px solid #DFE2EE;}
	#cpeConfigPage .cpeRadioGroup .el-radio-button__inner { padding: 0 20px; line-height: 26px; font-size: 12px; background: #FFFFFF; }
	#cpeConfigPage .cpeRadioGroup .el-radio-button:first-child .el-radio-button__inner { border-left:1px solid #DFE2EE; }
	#cpeConfigPage .cpeRadioGroup .el-radio-button__orig-radio:checked+.el-radio-button__inner { color: #333333; background-color: #E4F1FF; border-color: #4D84FF; }
    #cpeConfigPage .lanWarp .el-form-item { width: 20%; }
	/* 参数配置 */
	#cpeConfigPage .formContentWarp .el-form-item { width: 30%; }
	#cpeConfigPage .titleWarp { display: flex; margin-bottom: 22px; }
	#cpeConfigPage .titleWarp span { font-size: 14px; color: #333333; font-weight: bold; }
	#cpeConfigPage .cpeCircleTip { width: 7px; height: 7px; background: #333; border-radius: 50%; margin: 5px 6px 0 0; }
	#cpeConfigPage .formContentWarp { display: flex; margin-left: 14px; }
	#cpeConfigPage .el-form-item .el-switch { margin-top: 4px; }
	#cpeConfigPage .addErrorTip { color: #FA5555; font-size: 12px; }
	#cpeConfigPage .ipListTitle { position: relative; height: 40px; line-height: 40px; margin-left: 14px; justify-content: space-between;}
    #cpeConfigPage .systemWarp .el-form-item .el-form-item__label { width: 200px; margin-right: 0; margin-left: 14px; }
	#cpeConfigPage .systemWarp .el-form-item .el-form-item__content { margin-left: 200px; }
	#cpeConfigPage .lteWidthWarp .el-form-item { margin-bottom: 22px; }
	#cpeConfigPage .lteWidthWarp .el-form-item .el-form-item__label { width: 120px; }
    #cpeConfigPage .lteWidthWarp .el-form-item .el-form-item__content { display: inline-block; }
	#cpeConfigPage .earfcn-pci-line { display: flex; align-items: center; }
	#cpeConfigPage .lanWarp .el-form-item .el-form-item__label { margin-right: 30px; }
	#cpeConfigPage .systemWarp .el-form-item,
	#cpeConfigPage .lanWarp .el-form-item { margin-bottom: 22px; }
	#cpeConfigPage .systemWarp .el-form-item__error { margin-left: 15px; }
	#cpeConfigPage .selectScanMode .el-input__inner { height: 28px !important; width: 200px; }
	#cpeConfigPage .selectOptions .el-input { height: 28px !important; }
	#cpeConfigPage .selectOptions .el-input__inner { height: 28px !important; }
    #cpeConfigPage .downloadBtn:before,
    #cpeConfigPage .addIpsBtn:before { color: #7A7992; }
    #cpeConfigPage .commonFormFotter { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
    #cpeConfigPage .el-input.is-disabled .el-input__inner { height: 26px !important; }
    #cpeConfigPage .el-form-item__error { padding-top: 0px;}
    #cpeConfigPage .contentHeight { height: calc(100% - 100px); overflow: scroll; }
    #cpeConfigPage .newIconBoxCls-bt .el-icon-circle-close:before {
		content: '\e778';
		font-size: 12px;
	}
	#cpeConfigPage .el-tabs--top .el-tabs__item.is-top:last-child { border-left: 1px solid #E9EDF9; }
</style>

<div id="cpeConfigPage" style='background: #F6F7FB'>
   <div class='commonFlex commonContent' style='height: 99.8%; width:100%; '>
   		<!-- 左侧主体内容 -->
   		<div class='leftWarp'>
   			<div class='commonFlex commonContent titleBox commonTitle commonBorderBottom'>
   				<span class='AddTitle commonText14'>{{policyTitle}}</span>
   				<div class="newIconBoxCls-bt" style="right:15px;top:7px;" @click="cpeAddOrUpdateCancel">
					<span class="el-icon el-icon-circle-close"></span>
				</div>
   			</div>
   			<div class='mainContent' style='bottom: 70px; height: auto;'>
   				<el-form ref="addOrEditForm" :model="addOrEditForm" :rules="addOrEditRule" label-position="top" label-width="125px">			
                    <div class="basicInfoBox commonColor">
			        	<div class="group-title not-extend">
							<span class="title-icon"></span>
							<span class="title-text"><%=rb.getString("JiBenXinXi") %></span>
						</div>
						<!--  basic info --> 
						<div style='padding-left: 26px; padding-top: 16px;'>							
							<el-form-item label="<%=rb.getString("CeLueMingChen") %>" prop="policyName" class='inputCommon commonFlex' label-position="left">
				                <el-input v-model="addOrEditForm.policyName" :disabled="readonly" style="width: 300px;"></el-input>
				            </el-form-item>
				            <el-form-item label="<%=rb.getString("ChanPinXingHao") %>" prop="productModule" label-position="left" class='commonFlex inputCommon' style=" margin-bottom: 22px;">
				               <div class="model-selected" style='width: 1000px;'>
				                    <el-checkbox-group v-model="addOrEditForm.productModule" @change="productChange" :disabled="readonly">
				                        <el-checkbox-button v-for="item in productModels" :label="item" border>{{item}}</el-checkbox-button>
				                    </el-checkbox-group>
				                </div>
				            </el-form-item>
						</div>
					</div>
					
					<hr class="split-line">
					
					<!-- Software Upgrade,parameter config -->
					<div class='basicInfoBox'>
			        	<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 16px;'><%=rb.getString("CPEKeYiXuanZeYiXiaMoKuaiJinXingPeiZhi") %></span>
			        	<div style="padding-bottom: 6px;">
			        		<el-radio-group v-model='functionModulesSelect' class='selectMethodBox' @change='selectMethodChange'>
								<el-radio-button label="0" class='moduleConfigBox'>
									<i class='el-icon el-icon-status-upgrading commonIconStyle'></i>
			        				<span class='commonTextWeight14 methodTitle'><%=rb.getString("RuanJianShengJi") %></span>
								</el-radio-button>

								<el-radio-button label="2" class='moduleConfigBox'>
									<i class='el-icon el-icon-menu-system commonIconStyle'></i>
			        				<span class='commonTextWeight14 methodTitle'><%=rb.getString("ENBCanShuZiPeiZhi") %></span>
			        			</el-radio-button>							
							</el-radio-group>
			        	</div>
			        </div>
			        
			       	<!-- Software Upgrade -->
			        <div v-show="functionModulesSelect == '0'" class="basicInfoBox commonColor" style='margin-top: 20px;'>
			            <div class="group-title not-extend">
							<span class="title-icon"></span>
							<span class="title-text"><%=rb.getString("RuanJianShengJi") %></span>
							<el-switch v-model="addOrEditForm.upgradeEnable" :active-value="true" :inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="readonly"></el-switch>
						</div>
						<div style='padding-left: 26px;'>
				            <span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 20px; margin-top: 14px;'><%=rb.getString("NinKeYiShouDongHuoLieBiaoXuanZeChuShiBanBenHao") %></span>
				            <el-form-item label="<%=rb.getString("MuBiaoBanBen") %>" label-position="left" prop="targetVersion" class="selectVersionBox commonFlex inputCommon">
			                    <el-select v-model="addOrEditForm.targetVersion" :disabled="readonly" placeholder="<%=rb.getString("QingXuanZe") %>"> 
			                     	<el-option v-for="item in tarVersionList" :label="item.targetVersion" :value="item.id+''"></el-option>
			                    </el-select>
			                </el-form-item>
				            <div class='commonFlex originalBox'>
				            	<span class='commonSize14' style='width: 150px;'><%=rb.getString("ChuShiBanBen") %></span>
				                <el-checkbox v-model="addOrEditForm.specifyVersionType" true-label="1" false-label="0" :disabled="readonly" style='padding: 2px 6px 0 0;'></el-checkbox>
				            	<span class='commonSize12ExportText'><%=rb.getString("SuoYouAny") %></span>
				            </div>  
							<div class='commonFlex' style='padding: 10px 0 4px;'>
								<!-- 无选择的版本 -->
								<div class='addVersionWarp' v-show='addVersionBtnShow'>
									<div v-if="addOrEditForm.specifyVersionType == '1' || readonly == true">
										<i :class='addOrEditForm.specifyVersionType == "0" ? "defaultClass":"disabledClass"' class='el-icon el-icon-plus addVersionBtn commonDisplayBlock'></i>
									</div>
									<div v-else>
										<i @click='addVersionClick' :class='addOrEditForm.specifyVersionType == "0" ? "defaultClass":"disabledClass"' class='el-icon el-icon-plus addVersionBtn commonDisplayBlock'></i>
									</div>
									<span class='commonTitle12'><%=rb.getString("NinKeYiTianJiaYuanShiBanBen") %></span>
								</div> 
								<!-- 已选择或已手动添加的版本@query="originalVersionQuery" -->
								<div class='addVersionWarp' v-show='resultOriginalVersionShow'>
									<el-ctable ref="resultOriginalVersionTable" row-key="orginalVersion" class='commonBorderRadius' style='margin-bottom: 10px;'
									:data="resultOriginalVersionList.filter(item=>{
											return item.orginalVersion.indexOf(searchValue) > -1
									 })" :pagination="false" :rownumber="false">
										<div slot="toolbar">
											<div class='commonFlex commonToolBarBox selectedVersion' style='position: relative;'>
												<div class='queryGroup' style='margin-right: 6px;'> 
													<el-input v-model="searchValue" class='pairgrid-query' placeholder='<%=rb.getString("ChuShiBanBen")%>' style='width: 220px;'></el-input>
													<i class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
												</div>
												<div class='commonFlex' style='position: absolute; right: 0;' v-show="curType != 'view' && addOrEditForm.specifyVersionType == '0'">
													<div class='operBtn' @click='clearVersionBtnClick' style='margin-right: 10px;'><i class='el-icon el-icon-operation-clear commonTextNormal12'></i></div>
													<div class='operBtn' @click='addVersionClick' style='margin-right: 10px;'><i class='el-icon el-icon-plus commonTextNormal12'></i></div>
												</div>
											</div>
										</div>
 										<el-table-column label="<%=rb.getString("ChuShiBanBen") %>" prop="orginalVersion" show-overflow-tooltip="true">
											<template slot-scope="scope">
												<div class='table-suffix'>
							                        	<span class="text">{{scope.row.orginalVersion}} </span>
							                        	<span v-show="curType != 'view' && addOrEditForm.specifyVersionType=='0'"><i class='el-icon el-icon-circle-close ipTextWarp deleteVersion' @click='deleteVersionItem(scope.row)'></i></span>
							                        </div>
											</template>
										</el-table-column>
									</el-ctable>
								</div>
							</div> 
							<el-form-item label-width='0' prop="orginalVersion" style='margin-left: 150px;'>
			                	<el-input v-model="addOrEditForm.orginalVersion" v-show=false></el-input>
			                </el-form-item> 							
						</div>        
			        </div>
			        <!-- Param Configuration -->
                 	<div v-show="functionModulesSelect == '2'" class="basicInfoBox commonColor" style='margin-top: 20px;'>
			            <div class="group-title not-extend" style='position: relative;'>
							<div class='commonFlex'>
								<span class="title-icon"></span>
								<span class="title-text"><%=rb.getString("ENBCanShuZiPeiZhi") %></span>
								<el-switch v-model="addOrEditForm.configEnable" :active-value="true" :inactive-value="false" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="readonly"></el-switch>
								<!-- 执行修改后，OMC与 CPE交互较慢，导致用户造成错觉，页面提示 -->
								<div style='padding: 3px 0 0 10px;' class='commonTitle12'>
									<i class="el-icon el-icon-circle-info" style='margin-right: 6px;'></i><%=rb.getString("CaoZuoKeHuTiShi")%>
								</div>
							</div>
							<div style='position: absolute; display: flex; right: 30px;'>
								<!-- 导入，导出 -->
								<div v-show="AddBtnShow" class='licenseImportBox' @click='importFileBtn'>
									<i class='el-icon el-icon-operation-import importIcon'></i>
								</div>
								<div v-show="AddBtnShow" class='licenseImportBox' @click='exportFileBtn'>
									<i class='el-icon el-icon-operation-export importIcon'></i>
								</div>
							</div>
						</div>
			            <div style='padding-left: 26px; padding-top: 16px; '>
                            <div class='reginDeployingBox infoSpecifiedDevice' style=' width: 98%; padding-bottom: 100px;'>
                                 <el-tabs v-model="parameterConfigActive" @tab-click="clickTab" type='border-card' style='height: auto' >
                                    <el-tab-pane label="<%=rb.getString("WangLuoSheZhi") %>" name="network" style="position:relative">
                                        <div class='paramTwoBox'>
                                        	<!-- WLAN  -->
						 					<div>
							 					<div class="titleWarp">
							 						<div class="cpeCircleTip"></div>
							 						<span>WLAN</span>
							 					</div>
							 					<div style="display: none;">
													<el-form-item prop="wifiSsid"></el-form-item>
													<el-form-item prop="wifiEncryption"></el-form-item>
													<el-form-item prop="wifiPassphrase"></el-form-item>
													<el-form-item prop="wifi1Enable"></el-form-item>
													<el-form-item prop="wifi1Ssid"></el-form-item>
													<el-form-item prop="wifi1Encryption"></el-form-item>
													<el-form-item prop="wifi1Passphrase"></el-form-item>
													<el-form-item prop="wifi2Enable"></el-form-item>
													<el-form-item prop="wifi2Ssid"></el-form-item>
													<el-form-item prop="wifi2Encryption"></el-form-item>
													<el-form-item prop="wifi2Passphrase"></el-form-item>
													<el-form-item prop="wifi3Enable"></el-form-item>
													<el-form-item prop="wifi3Ssid"></el-form-item>
													<el-form-item prop="wifi3Encryption"></el-form-item>
													<el-form-item prop="wifi3Passphrase"></el-form-item>
												</div>	 					
							 					<div class="formContentWarp" >
							 						<el-form-item label="WiFi" prop="wlanEnable">
														<el-select v-model="addOrEditForm.wlanEnable" :disabled="readonly" class="selectOptions" @change="wlanEnableChange" placeholder="<%=rb.getString("QingXuanZe") %>"> 
															<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
														</el-select>
													</el-form-item>
													<el-form-item label="Frequency(Channel)" prop="wifiChannel">
														<el-select v-model="addOrEditForm.wifiChannel" :disabled="readonly" size="mini" placeholder="<%=rb.getString("QingXuanZe") %>">
															<!-- <el-option label="" value=""></el-option> -->
															<el-option label="AUTO" value="AUTO"></el-option>
															<el-option label="2.412GHz(Channel 1)" value="1"></el-option>
															<el-option label="2.417GHz(Channel 2)" value="2"></el-option>
															<el-option label="2.422GHz(Channel 3)" value="3"></el-option>
															<el-option label="2.427GHz(Channel 4)" value="4"></el-option>
															<el-option label="2.43GHz(Channel 5)" value="5"></el-option>
															<el-option label="2.437GHz(Channel 6)" value="6"></el-option>
															<el-option label="2.442GHz(Channel 7)" value="7"></el-option>
															<el-option label="2.447GHz(Channel 8)" value="8"></el-option>
															<el-option label="2.452GHz(Channel 9)" value="9"></el-option>
															<el-option label="2.457GHz(Channel 10)" value="10"></el-option>
															<el-option label="2.462GHz(Channel 11)" value="11"></el-option>
														</el-select>
													</el-form-item>
													
													<el-form-item label="Channel Bandwidth" prop="wifiBandwidth" style='margin-bottom: 30px;'>
										                <el-radio-group v-model="addOrEditForm.wifiBandwidth" :disabled="readonly" class='cpeRadioGroup'>
										                    <el-radio-button label="0">20M</el-radio-button>
										                    <el-radio-button label="1">20/40M</el-radio-button>
										                </el-radio-group>
										            </el-form-item>
							 					</div>
							 					
							 					<div style="margin-left: 14px; margin-bottom: 22px;">		 					
							 						<div style="margin-bottom: 10px; font-size: 14px; ">MBSSID</div>
													<el-ctable height="177" :data="tbData" :pagination="false" style="border: 1px solid #E9E9E9; width:80%;" :disabled="readonly">
														<el-table-column width="40">
															<template slot-scope="scope">
																<i class="el-icon el-icon-operation-edit commonIcon" @click="modifyWifi(scope.row)" v-show="!isWifiClosed && editBtnShow"></i>
															</template>
														</el-table-column>
														<el-table-column label="Network Name(SSID)" prop="wifiSsid"></el-table-column>
														<el-table-column label="Security Mode" prop="wifiEncryption">
															<template slot-scope="scope">
																{{modeKeys[scope.row.wifiEncryption]}}
															</template>
														</el-table-column>
														<el-table-column label="Status" prop="wifiEnable">
															<template slot-scope="scope">
																<span v-if="scope.row.wifiEnable=='1'">Enable</span>
																<span v-if="scope.row.wifiEnable=='0'">Disable</span>
															</template>
														</el-table-column>									
													</el-ctable>
							 					</div>
						 						<!-- wifi5 -->
						 						<div style="display: none;">
													<el-form-item prop="wifi5WifiSsid"></el-form-item>
													<el-form-item prop="wifi5WifiEncryption"></el-form-item>
													<el-form-item prop="wifi5WifiPassphrase"></el-form-item>
						
													<el-form-item prop="wifi5Wifi1Enable"></el-form-item>
													<el-form-item prop="wifi5Wifi1Ssid"></el-form-item>
													<el-form-item prop="wifi5Wifi1Encryption"></el-form-item>
													<el-form-item prop="wifi5Wifi1Passphrase"></el-form-item>
						
													<el-form-item prop="wifi5Wifi2Enable"></el-form-item>
													<el-form-item prop="wifi5Wifi2Ssid"></el-form-item>
													<el-form-item prop="wifi5Wifi2Encryption"></el-form-item>
													<el-form-item prop="wifi5Wifi2Passphrase"></el-form-item>
						
													<el-form-item prop="wifi5Wifi3Enable"></el-form-item>
													<el-form-item prop="wifi5Wifi3Ssid"></el-form-item>
													<el-form-item prop="wifi5Wifi3Encryption"></el-form-item>
													<el-form-item prop="wifi5Wifi3Passphrase"></el-form-item>
												</div>
												<div class="formContentWarp">
						 							<el-form-item label="WiFi5" prop="wifi5WlanEnable">
														<el-select v-model="addOrEditForm.wifi5WlanEnable" :disabled="readonly" class="selectOptions" @change="wifi5EnableChange"> 
															<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
														</el-select>
													</el-form-item>
													<el-form-item label="Network Mode" prop="wifi5WifiMode">
														<el-select v-model="addOrEditForm.wifi5WifiMode">
															<el-option label="11a" value="AONLY"></el-option>
															<el-option label="11a/n" value="AN"></el-option>
															<el-option label="11a/n/ac" value="A_AN_AC"></el-option>
															<el-option label="11an/ac" value="AN_AC"></el-option>
															<el-option label="11a/n/ac/ax" value="A_AN_AC_AX"></el-option>
														</el-select>
													</el-form-item>
													<el-form-item label="Channel" prop="wifi5WifiChannel">
														<el-select v-model="addOrEditForm.wifi5WifiChannel" size="mini">
															<el-option label="AUTO" value="AUTO"></el-option>
															<el-option v-for="item in wifi5Chanel[addOrEditForm.wifi5WifiSupportChannel]" :label="item" :value="item"></el-option>
														</el-select>
													</el-form-item>
						 						</div>
						 						<div class="formContentWarp">
						 							<el-form-item label="Channel Bandwidth" prop="wifi5WifiBandwidth" style='margin-bottom: 30px;'>
										                <el-radio-group v-model="addOrEditForm.wifi5WifiBandwidth" :disabled="readonly" class='cpeRadioGroup'>
										                    <el-radio-button label="0">20M</el-radio-button>
										                    <el-radio-button label="1">40M</el-radio-button>
										                    <el-radio-button label="2">80M</el-radio-button>
										                    <el-radio-button label="3">160M</el-radio-button>
										                </el-radio-group>
										            </el-form-item>
						 							
													<el-form-item label="Support Channel" prop="wifi5WifiSupportChannel">
														<el-select v-model="addOrEditForm.wifi5WifiSupportChannel" size="mini" @change="function(val){addOrEditForm.wifi5WifiChannel = '';}">
															<el-option value="SC1" label="Ch36~48"></el-option>
															<el-option value="SC2" label="Ch36~64"></el-option>
															<el-option value="SC3" label="Ch52~64"></el-option>
															<el-option value="SC4" label="Ch149~161"></el-option>
															<el-option value="SC5" label="Ch149~165"></el-option>
															<el-option value="SC6" label="Ch36~48,Ch149~161"></el-option>
															<el-option value="SC7" label="Ch36~48,Ch149~165"></el-option>
															<el-option value="SC8" label="Ch36~64,Ch100~140"></el-option>
															<el-option value="SC9" label="Ch36~64,Ch149~161"></el-option>
															<el-option value="SC10" label="Ch52~64,Ch149~161"></el-option>
															<el-option value="SC11" label="Ch52~64,Ch149~165"></el-option>
															<el-option value="SC12" label="Ch36~64,Ch100~120,Ch149~161"></el-option>
															<el-option value="SC13" label="Ch36~64,Ch100~116,Ch132~140"></el-option>
															<el-option value="SC14" label="Ch36~64,Ch100~124,Ch149~161"></el-option>
															<el-option value="SC15" label="Ch36~64,Ch100~140,Ch149~161"></el-option>
															<el-option value="SC16" label="Ch36~64,Ch100~140,Ch149~165"></el-option>
															<el-option value="SC17" label="Ch52~64,Ch100~140,Ch149~161"></el-option>
															<el-option value="SC18" label="Ch56~64,Ch100~140,Ch149~161"></el-option>
															<el-option value="SC19" label="Ch36~64,Ch100~116,Ch132~140,Ch149~165"></el-option>
															<el-option value="SC20" label="Ch36~64,Ch100~116,Ch136~140,Ch149~165"></el-option>
														</el-select>
													</el-form-item>
						 						</div>
						 						
						 						<div style="margin-left: 14px; margin-bottom: 22px;">		 					
							 						<div style="margin-bottom: 10px; font-size: 14px; ">MBSSID</div>
													<el-ctable height="177" ref="taskList" :data="tbDataWifi5" :pagination="false" style="border: 1px solid #E9E9E9;width:80%; ">
														<el-table-column width="80">
															<template slot-scope="scope">
																<i class="el-icon el-icon-operation-edit" @click="modifyWifi5(scope.row)"  v-show="!isWifi5Closed && editBtnShow"></i>
															</template>
														</el-table-column>
														<el-table-column label="Network Name(SSID)" prop="wifi5WifiSsid"></el-table-column>
														<el-table-column label="Security Mode" prop="wifi5WifiEncryption">
															<template slot-scope="scope">
																{{modeKeys[scope.row.wifi5WifiEncryption]}}
															</template>
														</el-table-column>
														<el-table-column label="Status" prop="wifi5WifiEnable">
															<template slot-scope="scope">
																<span v-if="scope.row.wifi5WifiEnable=='1'">Enable</span>
																<span v-if="scope.row.wifi5WifiEnable=='0'">Disable</span>
															</template>
														</el-table-column>									
													</el-ctable>
							 					</div>
						 					
						 					</div>
						 					<!--DMZ  -->	
						 					<div>
							 					<div class="titleWarp" style='margin-bottom: 22px; '>
							 						<div class="cpeCircleTip"></div>
							 						<span>DMZ</span>
							 					</div>				 							 					
							 					<div class="formContentWarp">
													<el-form-item label="DMZ Enable" prop="dmzEnable">
														<el-select v-model="addOrEditForm.dmzEnable" :disabled="readonly" class="selectOptions" placeholder="<%=rb.getString("QingXuanZe") %>"> 
															<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
														</el-select>
													</el-form-item>
													<el-form-item label="DMZ Address" prop="dmzHostAddress" >
														<el-input v-model="addOrEditForm.dmzHostAddress" maxlength="100" :disabled="readonly"></el-input>
													</el-form-item>
							 					</div>
						 					</div>	
						 					<!--LAN  -->	
						 					<div>
							 					<div class="titleWarp" style='margin-bottom: 22px; '>
							 						<div class="cpeCircleTip"></div>
							 						<span>LAN</span>
							 					</div>			 							 					
							 					<div style='margin-left: 14px;' class="lanWarp ">
													<el-form-item label="LAN <%=rb.getString("JieKou")%>" prop="lanEnable">
														<el-select v-model="addOrEditForm.lanEnable" :disabled="readonly" class="selectOptions" placeholder="<%=rb.getString("QingXuanZe") %>"> 
															<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
														</el-select>
													</el-form-item>
							 					</div>
							 					<!--DHCP Settings  -->
							 					<div class="formContentWarp">
							 						<el-form-item label="DHCP Server" prop="lanDHCPEnable">
														<el-select v-model="addOrEditForm.lanDHCPEnable" :disabled="readonly" class="selectOptions" placeholder="<%=rb.getString("QingXuanZe") %>"> 
															<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
														</el-select>
													</el-form-item>
													<el-form-item label="DNS Option" prop="lanOption">
														<el-radio-group v-model="addOrEditForm.lanOption" :disabled="readonly">
															<!--原：Auto 自动 : 0; 手动： 1-->
										                    <el-radio :label="'1'" @click.native='clickLanOption($event)' border><%=rb.getString("CPEZiDong")%></el-radio>
										                    <el-radio :label="'0'" @click.native='clickLanOption($event)' border><%=rb.getString("CPEShouDong")%></el-radio>
										                </el-radio-group>
													</el-form-item>
													<el-form-item label="DNS1" prop="lanDNS1" v-if='addOrEditForm.lanOption == "0"'>
														<el-input v-model="addOrEditForm.lanDNS1" :disabled="readonly"></el-input>
													</el-form-item>
													<el-form-item label="DNS2" prop="lanDNS2" v-if='addOrEditForm.lanOption == "0"'>
														<el-input v-model="addOrEditForm.lanDNS2" :disabled="readonly"></el-input>
													</el-form-item>
							 					</div>
						 					</div>
										</div>
                                    </el-tab-pane>
                                    <!-- LTE -->
                                    <el-tab-pane label="LTE" name="lte">
                                        <div class='paramTwoBox'>
                                            <!-- PCI Lock  -->
						 					<div>
							 					<div class="titleWarp">
							 						<div class="cpeCircleTip"></div>
							 						<span><%=rb.getString("SuoPCI")%></span>
							 					</div>				 							 					
							 					<div style="margin-left: 14px;" class="lteWidthWarp">
							 						<el-form-item label="<%=rb.getString("SaoMiaoFangShi")%>" prop="scanMode">
														<el-select v-model="addOrEditForm.scanMode" :disabled="readonly" class="selectScanMode" placeholder="<%=rb.getString("QingXuanZe") %>"> 
															<!--  <el-option label="" value=""></el-option>-->
															<el-option label="Full Band" value="fullband"></el-option>
															<el-option label="Band/Frequency Preferred" value="freqpreferred"></el-option>
															<el-option label="PCI lock" value="pcilock"></el-option>
															<el-option label="PCI Only Lock" value="pcionlylock"></el-option>
														</el-select>
													</el-form-item>	
													<el-form-item v-show="false" prop="pci">
														<el-input v-model="addOrEditForm.pci"></el-input>
													</el-form-item>	
													<!-- earfcn -->
													<div v-if="addOrEditForm.scanMode == 'freqpreferred'">
														<el-form-item class="earfcn-pci-line" label='<%=rb.getString("ENBPinDian")%>' style="margin-bottom: 0px;">
															<div class="mainWarp">						
																<div class='newAddBtn'>
																	<el-input v-model='addOrEditForm.onlyEarfcn' :disabled="readonly"></el-input>
																	<span v-show="AddBtnShow" @click='addEarfcnBtn' class='form-bt el-icon el-icon-plus addIpsBtn' style='vertical-align:middle'></span> 
																</div>							
															</div>
														</el-form-item>
														<div class="versionResultBox" v-show='addOrEditForm.earfcnList.length > 0' style='margin-left: 120px;'>
															<el-form-item class='suffixItem' v-for='(domain,index) in addOrEditForm.earfcnList' style='margin-bottom: 0 !important;'>
																<div class='form-suffix'>
																	<span class='text'><%=rb.getString("ENBPinDian")%> : {{domain}}</span>
																	<span v-show="AddBtnShow" class='form-bt-remove el-icon el-icon-operation-delete ipTextWarp deleteVersion' @click.prevent='removeEarfcn(domain)'></span>
																</div>
															</el-form-item> 
														</div>	
														<p class="addErrorTip" style='margin-left: 120px;'>{{earfcnErrorMessage}}</p> 										
													</div>
													<!-- earfcn and pci -->
													<div v-if="addOrEditForm.scanMode == 'pcilock'">
														<el-form-item class="earfcn-pci-line" label='<%=rb.getString("ENBPinDian")%> : PCI' style="margin-bottom: 0px;">
															<div class="mainWarp">						
																<div class='newAddBtn'>
																	<el-input v-model='addOrEditForm.earfcnStart' style='width:200px;' :disabled="readonly"></el-input> — <el-input v-model='addOrEditForm.pciEnd' :disabled="readonly" style='width:200px;'></el-input>		 																
																	<span v-show="AddBtnShow" @click='addPcilockBtn' class='form-bt el-icon el-icon-plus addIpsBtn' style='vertical-align:middle'></span> 
																</div>							
															</div>
														</el-form-item>
														<div class="versionResultBox" v-show='addOrEditForm.earfcnPciList.length > 0' style='margin-left: 120px;'>
															<el-form-item class='suffixItem' v-for='(domain,index) in addOrEditForm.earfcnPciList' style='margin-bottom: 0 !important;'>
																<div class='form-suffix'>
																	<div style="display: flex;">
																		<span class='text'><%=rb.getString("ENBPinDian")%> : {{domain.startValue}}</span>
																		<span class='text' style="margin: 0 10px 0 15px;">PCI : {{domain.endValue}}</span>
																		<span class='form-bt-remove el-icon el-icon-operation-delete ipTextWarp deleteVersion' @click.prevent='removeEarfcnPci(domain)'></span>															
																	</div>														 
																</div>
															</el-form-item> 
														</div>	
														<p class="addErrorTip" style='margin-left: 120px;'>{{earfcnPciErrorMessage}}</p>										
													</div>
													<!--  pci -->
													<div v-if="addOrEditForm.scanMode == 'pcionlylock'">
														<el-form-item class="earfcn-pci-line" label='PCI' style="margin-bottom: 0px;">
															<div class="mainWarp">						
																<div class='newAddBtn'>
																	<el-input v-model='addOrEditForm.onlyPci' :disabled="readonly"></el-input>
																	<span v-show="AddBtnShow" @click='addPciBtn' class='form-bt el-icon el-icon-plus addIpsBtn' style='vertical-align:middle'></span> 
																</div>							
															</div>
														</el-form-item>
														<div class="versionResultBox" v-show='addOrEditForm.pciList.length > 0' style='margin-left: 120px;'>
															<el-form-item class='suffixItem' v-for='(domain,index) in addOrEditForm.pciList' style='margin-bottom: 0 !important;'>
																<div class='form-suffix'>
																	<span class='text'>PCI : {{domain}}</span>
																	<span class='form-bt-remove el-icon el-icon-operation-delete ipTextWarp deleteVersion' @click.prevent='removePci(domain)'></span>
																</div>
															</el-form-item> 
														</div>
														<p class="addErrorTip" style='margin-left: 120px;'>{{pciErrorMessage}}</p>
													</div>																	
							 					</div>				 									 					
						 					</div>
                                        </div>
                                    	<!-- apn 提示 -->
										<div style='padding: 10px 0 0 32px;'>
											<i class="el-icon el-icon-circle-info commonIcon" style='margin-right: 6px; font-size: 14px;'></i><%=rb.getString("APNPeiZhiTiShi")%>
										</div>
                                    </el-tab-pane>
                                    
                                    <!-- system-->
                                    <el-tab-pane label="<%=rb.getString("XiTong")%>" name="system">
                                        <div class='paramTwoBox'>
                                        	<!-- Web Access  -->
						 					<div> 
							 					<div class="titleWarp">
							 						<div class="cpeCircleTip"></div>
							 						<span>Web Access</span>
							 					</div>
							 					<div style='margin-left: 14px;' class="lanWarp commonFlex">
													<el-form-item label="Password" prop="uiPassword">
														<el-input v-model="addOrEditForm.uiPassword" maxlength="100" :disabled="readonly"></el-input>
													</el-form-item>
													<el-form-item label="Https<%=rb.getString("SheZhiKaiGuan")%>" prop="httpsEnable">
														<el-select v-model="addOrEditForm.httpsEnable" :disabled="readonly" class="selectOptions" placeholder="<%=rb.getString("QingXuanZe") %>"> 
															<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
														</el-select>
													</el-form-item>
													<el-form-item label="HttpsWan<%=rb.getString("SheZhiKaiGuan")%>" prop="httpsWanEnable">
														<el-select v-model="addOrEditForm.httpsWanEnable" :disabled="readonly" class="selectOptions" placeholder="<%=rb.getString("QingXuanZe") %>"> 
															<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
														</el-select>
													</el-form-item>
													<el-form-item label="Access Control <%=rb.getString("SheZhiKaiGuan")%>" prop="wanAccEnable">
														<el-select v-model="addOrEditForm.wanAccEnable" :disabled="readonly" class="selectOptions" placeholder="<%=rb.getString("QingXuanZe") %>"> 
															<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
														</el-select>
													</el-form-item>
													<el-form-item v-show="false" prop="rangeIp">
														<el-input v-model="addOrEditForm.rangeIp"></el-input>
													</el-form-item>	
							 					</div>		 					
							 					<div class="systemWarp">
													<div class="ipListTitle commonFlex" style="width: 80%; margin-top: -10px;">
														<span style="font-size: 14px; color: #333333;"><%=rb.getString("FangWenKongZhiLieBiao")%></span>
														<div v-show="AddBtnShow" class='operBtn' @click='systemAddIp'><i class='el-icon el-icon-plus commonTextNormal12'></i></div>
													</div>	
													<div style="margin-bottom:30px;margin-left: 14px; width:80%;">
														<el-ctable ref="ctableIp" :data="ipTableData" height="200":pagination="false" :rownumber="true" class='commonBorderRadius commonBorder'>
s															<el-table-column label='' width="70" >
																<template slot-scope="scope">
																	<div v-show="AddBtnShow">
																		<i v-show="ipTableData.length > 0 && curType != 'view'" @click="updateIp(scope.row, scope.$index)" class="el-icon el-icon-operation-edit commonIcon"></i>
																		<i v-show="ipTableData.length > 0 && curType != 'view'" @click="deleteIp(scope.row, scope.$index)" class="el-icon el-icon-operation-delete commonIcon" style='margin-left: 6px;'></i>
																	</div>
																</template>
															</el-table-column>												
															<el-table-column label="<%=rb.getString("KaiShiIP") %>" prop="ipStart"></el-table-column>
															<el-table-column label="<%=rb.getString("JieShuIP") %>" prop="ipEnd"></el-table-column>
														</el-ctable>
													</div>							
							 					</div>					 								 									 					
						 					</div>
						 					<!-- Ping Watchdog  -->
						 					<div>
							 					<div class="titleWarp">
							 						<div class="cpeCircleTip"></div>
							 						<span>Ping Watchdog</span>
							 					</div>				 							 					
							 					<div class="lanWarp commonFlex" style='margin-left: 12px;'>
							 						<el-form-item label="Ping Watchdog <%=rb.getString("KaiGuan")%>" prop="watchDogEnable">
														<el-select v-model="addOrEditForm.watchDogEnable" :disabled="readonly" class="selectOptions" placeholder="<%=rb.getString("QingXuanZe") %>"> 
															<el-option v-for="item in selectOptions" :label="item.text" :value="item.value"></el-option>
														</el-select>
													</el-form-item>
													<el-form-item label="IP Address or URL to Ping" prop="watchDogPingIp" >
														<el-input v-model="addOrEditForm.watchDogPingIp" maxlength="45" placeholder="Length: 0-45" :disabled="readonly"></el-input>
													</el-form-item>
													<el-form-item label="Ping Timeout(Seconds)" prop="watchDogPingTimeout">
														<el-input v-model="addOrEditForm.watchDogPingTimeout" maxlength="8" placeholder="Range: 1-65535" :disabled="readonly"></el-input>
													</el-form-item>
													<el-form-item label="Ping Count" prop="watchDogPingCount">
														<el-input v-model="addOrEditForm.watchDogPingCount" maxlength="8" placeholder="Range: 1-65535" :disabled="readonly"></el-input>
													</el-form-item>
													<el-form-item label="Failure Count to Reboot" prop="watchDogFailureReboot">
														<el-input v-model="addOrEditForm.watchDogFailureReboot" maxlength="8" placeholder="Range: 1-65535" :disabled="readonly"></el-input>
													</el-form-item>									
							 					</div>				 									 					
						 					</div>						 				
                                        </div>
                                    </el-tab-pane>
                                </el-tabs>                              
                            </div>
                         </div>
                	</div>
                </el-form>   				
   			</div>
   			
		    <div class='commonFlex commonContent footerBox commonBorderTop' v-show="curType !='view'">
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" size="mini" @click='cpeAddOrUpdateSubmit'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='cpeAddOrUpdateCancel'><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>            
   		
   		<!--  右侧 新建，修改，详情，导入等内容展示-->
   		<!--software upgrade: Add Version  -->
   		<div class='rightBox' style='position: relative;' v-show='addVersionShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("TianJianBanBen")%></span>
   				<span class='closeIconBox' @click='addVersionCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 10px 0 20px;'>
   				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'><%=rb.getString("ShouDongHuoXuanZeBanBen")%></span></div>
   				<div class='originalVersionBox'>
   				   	<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 6px; '><%=rb.getString("ChuShiBanBen") %></span>
   				
   					<el-form ref="softwareAddVersionForm" :inline="true" label-position="top" :model="softwareAddVersionForm" :rules="softwareAddVersionRules" style="display: flex; flex-direction: column;overflow: hidden;">
	   					<el-form-item style='margin-bottom: 16px;'>
							<el-form-item prop='versionStr'>
								<el-input v-model='softwareAddVersionForm.versionStr'>
									 <i slot="append" class="el-icon el-icon-plus" @click="addVersionBtn"></i>
								</el-input>						
							</el-form-item>
								
							<div class='versionResultBox' v-show='softwareAddVersionForm.versionList.length > 0'>
								<el-form-item class='suffixItem' v-for='(domain,index) in softwareAddVersionForm.versionList'>
									<div class='form-suffix'>
										<span class='text'>{{domain}}</span>
										<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteVersion' @click.prevent='removeVersion(domain)'></span>
									</div>
								</el-form-item>
								<el-form-item prop='itemTest'>
									<el-input v-model='softwareAddVersionForm.itemTest' v-show=false></el-input>
								</el-form-item>
							</div>
							<p class='ipErrorTip'>{{versionErrorMessage}}</p>
	                    </el-form-item>       
   					</el-form>
   				</div>
   				
   				<div>
   				<!-- 初始版本列表 -->
	                <span class='commonTitle12 commonDisplayBlock'><%=rb.getString("ChuShiBanBenLieBiao")%></span>	                  
					<el-ctable ref="softwareOriginalVersionTable" height='280px' class='commonBorderRadius commonBorder' style='margin-top: 8px;' :pagination="false" :rownumber="false" 
						:data="softwareOriginalVersionList" row-key="software_version" @selection-change='versionBatchSelect'>
						<el-table-column type="selection" :reserve-selection="true"></el-table-column>
						<el-table-column label="<%=rb.getString("ChuShiBanBen") %>" prop="software_version" show-overflow-tooltip="true"></el-table-column>
					</el-ctable>
	            </div>
	            <p class='ipErrorTip' style='margin-top: 2px;'>{{versionMessage}}</p>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" size="mini" @click='addVersionSubmit'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='addVersionCancel'><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
   		
   		<!--param config: import  -->
   		<div class='rightBox' style='position: relative;' v-show='showImportCard'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("DaoRu")%></span>
   				<span class='closeIconBox' @click='closeImportParams'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='rightTextBox commonBorderBottom'>
   					<span class='commonTitle12'><%=rb.getString("DaoRuWenJianTiShi")%></span>
   					<div style="cursor: pointer; margin-top: 10px; margin-bottom: 6px;" @click="exportTemplate">
						<span class='el-icon el-icon-common-download downloadBtn'></span>
						<span class='commonTextNormal12' style='text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
					</div>
   				</div>
   				<div class='originalVersionBox'>
   					<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 6px; '><%=rb.getString("WenJian")%>(<span class='commonTitle12'><%=rb.getString("DangQianZhiChiWenJianLeiXing")%></span>)</span>
   					<el-form label-position="top" ref="importRuleForm" :model='importRuleForm' :rules='importRules'>     		     			            
			        	<el-form-item label=" " label-width="110px" prop="fileName">
			               	<el-upload 
			             		ref="upload"
			             		:before-upload='beforeUpload' 
			             		:on-success='checkFile' 
			             		:on-change="fileChange"  
			             		:show-file-list=false 	                  		
							    :action="importRuleForm.uploadFileUrl" 
							    :data="fileParams" 
							    name="uploadFile" 
							    accept=".xls,.xlsx"
							    :auto-upload="false">
								<el-input :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
									<a slot="append" class="el-icon el-icon-operation-import importBox" @click="fileSelect"></a>
								</el-input>								
								<a slot="trigger" ref="file_up"></a>
							</el-upload>	
			         	</el-form-item>      	    
			        </el-form> 
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="uploadParams"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="closeImportParams"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
   		<!--param config: add ip  -->
   		<div class='rightBox' style='position: relative;' v-show='addIpShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox commonBorderBottom' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("TianJia")%></span>
   				<span class='closeIconBox' @click='closeAddIp'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='originalVersionBox'>
   					<el-form ref='addIpForm' :model='addIpForm' label-position="top">
						<el-form-item label="IP" label-width="50px"  style='margin-bottom:0px;position:relative'>
							<el-input v-model='addIpForm.ipStart' style='width: 125px;'></el-input> — <el-input v-model='addIpForm.ipEnd' style='width: 125px;'></el-input>		 		
							<span @click='addIpBtn' class='form-bt el-icon el-icon-plus addIpsBtn' style='vertical-align:middle'></span> 
						</el-form-item>
						<div class="versionResultBox" v-show='addIpForm.ipGroup.length > 0' style='max-height: 300px;'>
							<el-form-item class='suffixItem' v-for='(domain,index) in addIpForm.ipGroup' style='margin-bottom: 0;'>
								<div class='form-suffix'>
									<span class='text' style='width: auto;'>{{domain}}</span>
									<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteVersion' @click.prevent='removeIp(domain)'></span>
								</div>
							</el-form-item> 
						</div>
						<p class="addErrorTip">{{errorMessage}}</p>
					</el-form> 
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="saveAddIp"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="closeAddIp"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
   		<!--param config: modify ip  -->
   		<div class='rightBox' style='position: relative;' v-show='editIpShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox commonBorderBottom' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("XiuGai")%></span>
   				<span class='closeIconBox' @click='closeEditIp'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='originalVersionBox'>
   					<el-form ref='editIpForm' :model='editIpForm' label-position="top">
						<el-form-item label="IP" label-width="50px"  style='margin-bottom:0px;position:relative'>
							<el-input v-model='editIpForm.ipStart' style='width: 140px;'></el-input> — <el-input v-model='editIpForm.ipEnd' style='width: 140px;'></el-input>		 		
							<p class="addErrorTip">{{editErrorMsg}}</p>
						</el-form-item>
					</el-form> 
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="saveEditIp"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="closeEditIp"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
   		<!--param config: modify mbssid  -->
   		<div class='rightBox' style='position: relative;' v-show='wifiMDLShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox commonBorderBottom' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("XiuGai")%> MBSSID</span>
   				<span class='closeIconBox' @click='cancelMbssidModify'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='originalVersionBox'>
   					<el-form ref="modifyForm" :model="mForm" :rules="mFormRules" label-position="top">
						<el-form-item v-show="false" prop="id">
							<el-input v-model="mForm.id"></el-input>
						</el-form-item>
						<el-form-item v-show="mForm.id != 'main'" label="Muti-SSID Status" prop="wifiEnable">
							<el-select v-model="mForm.wifiEnable">
								<el-option label="Enable" value="1"></el-option>
								<el-option label="Disable" value="0"></el-option>
							</el-select>
						</el-form-item>
						<div v-show="mForm.wifiEnable == '1'">
							<el-form-item label="Network Name(SSID)" prop="wifiSsid">
								<el-input v-model="mForm.wifiSsid" maxlength="100"></el-input>
							</el-form-item>
							<el-form-item label="Security Mode" prop="wifiEncryption">
								<el-select v-model="mForm.wifiEncryption">
									<el-option label="OPEN" value="OPEN"></el-option>
									<el-option label="WPAPSK" value="WPA"></el-option>
									<el-option label="WPA2PSK" value="WPA2"></el-option>
									<el-option label="WPAPSK/WPA2PSK" value="WPAWPA2"></el-option>
								</el-select>
							</el-form-item>
							<div v-show="mForm.wifiEncryption != 'open'">
								<el-form-item v-show="mForm.wifiEncryption && mForm.wifiEncryption!='open' && false" label="WPA Algorithm">
									{{wpakeys[mForm.wifiEncryption]}}
								</el-form-item>
								<div style="padding-bottom: 25px;">
									<el-checkbox v-model="mForm.showPassword">Display Password</el-checkbox>
									<el-form-item v-show="false" prop="showPassword"></el-form-item>
								</div>
								<el-form-item label="Pass Phrase" prop="wifiPassphrase">
									<el-input :type="mForm.showPassword==true?'text':'password'" v-model="mForm.wifiPassphrase"></el-input>
								</el-form-item>
							</div>
						</div>
					</el-form>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="saveMbssidModify"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="cancelMbssidModify"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
   		
   		<!-- wifi5 MBSSID修改窗口 -->
   		<div class='rightBox' style='position: relative;' v-show='wifi5WifiMDLShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox commonBorderBottom' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("XiuGai")%> MBSSID</span>
   				<span class='closeIconBox' @click='cancelWifiMbssidModify'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='originalVersionBox'>
   					<el-form ref="modifyWifi5Form" :model="mWifi5Form" :rules="mWifi5FormRules" label-position="top">
						<el-form-item v-show="false" prop="id">
							<el-input v-model="mWifi5Form.id"></el-input>
						</el-form-item>
						<el-form-item v-show="mWifi5Form.id != 'main'" label="Muti-SSID Status" prop="wifi5WifiEnable">
							<el-select v-model="mWifi5Form.wifi5WifiEnable">
								<el-option label="Enable" value="1"></el-option>
								<el-option label="Disable" value="0"></el-option>
							</el-select>
						</el-form-item>
						<div v-show="mWifi5Form.wifi5WifiEnable == '1'">
							<el-form-item label="Network Name(SSID)" prop="wifi5WifiSsid">
								<el-input v-model="mWifi5Form.wifi5WifiSsid" maxlength="100"></el-input>
							</el-form-item>
							<el-form-item label="Security Mode" prop="wifi5WifiEncryption">
								<el-select v-model="mWifi5Form.wifi5WifiEncryption">
									<el-option label="OPEN" value="OPEN"></el-option>
									<el-option label="WPAPSK" value="WPA"></el-option>
									<el-option label="WPA2PSK" value="WPA2"></el-option>
									<el-option label="WPAPSK/WPA2PSK" value="WPAWPA2"></el-option>
								</el-select>
							</el-form-item>
							<div v-show="mWifi5Form.wifi5WifiEncryption != 'open'">
								<el-form-item v-show="mWifi5Form.wifi5WifiEncryption && mWifi5Form.wifi5WifiEncryption!='open' && false" label="WPA Algorithm">
									{{wpakeys[mWifi5Form.wifi5WifiEncryption]}}
								</el-form-item>
								<div style="padding-bottom: 25px;">
									<el-checkbox v-model="mWifi5Form.wifi5ShowPassword">Display Password</el-checkbox>
									<el-form-item v-show="false" prop="wifi5ShowPassword"></el-form-item>
								</div>
								<el-form-item label="Pass Phrase" prop="wifi5WifiPassphrase">
									<el-input :type="mWifi5Form.wifi5ShowPassword==true?'text':'password'" v-model="mWifi5Form.wifi5WifiPassphrase"></el-input>
								</el-form-item>
							</div>
						</div>
					</el-form>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="saveModifyWifi5"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="cancelWifiMbssidModify"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
 		<!--param config  -->
   </div>   
</div>	

<script>
    var cpeAddOrEditCpeConfigVue = new Vue({
        el: '#cpeConfigPage',
        data() {
            var vm = this,
                nozhreg = '^(?:(?![\u4E00-\u9FA5]|[\uFE30-\uFFA0]).)*$',
                ipreg = '^(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|[1-9])\\.'
                        +'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)\\.'
                        +'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)\\.'
                        +'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)|%config$',
                              
	            validatePolicyName = function(rule,value,callback){			  
					if(value === '' || value === null || value === undefined) {
						callback(new Error('<%=rb.getString("BiTian")%>'));
					}else{
						callback();
					}
				},
				//原始版本
				validateVersion = function(rule, value, callback) {
					var verType = vm.addOrEditForm.specifyVersionType;
					//software upgrade 开关打开校验必填项
					if(vm.addOrEditForm.upgradeEnable == true){
	                    if(verType == '1') {
	                    	callback();
	                    }else if(vm.resultOriginalVersionList.length == 0) {
	                    	callback('<%=rb.getString("QingXuanZeJiLu") %>');
	                    }else{
	                    	callback();
	                    }
                    }else{
                    	callback();
                    }					
                },
                //目标版本
                validateTarget = function(rule,value,callback) {
                	//software upgrade 开关打开校验必填项
					if(vm.addOrEditForm.upgradeEnable == true){
	                	if( value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("BiTian")%>');
						}else {
							callback();
						}
					}else{
                    	callback();
                    }
				},
				//产品型号
				validateProModule = function(rule, value, callback) {
					if(vm.addOrEditForm.productModule.length == 0) {
                    	callback('<%=rb.getString("BiTian") %>');
                    }else{
                    	callback();
                    }
				},
				/* watchdog ip校验 */
				watchdogValidateIP = function(rule,value,cb) {
					var currVal = value, enable = vm.addOrEditForm.watchDogEnable;
					
					if(currVal && currVal.length>45){
						cb('<%=rb.getString("SheBeiMingChengGuiZe")%>');
					}else{
						if(!currVal && enable == 1) {
							cb('<%=rb.getString("SheBeiMingChengGuiZe")%>');
						}else {
							cb();
						}
					}
				},
			    /* watchdog相关校验 */
				watchdogValid = function(rule,value,cb){
					var currVal = value, enable = vm.addOrEditForm.watchDogEnable;
										
					if(currVal) {
						if(isNaN(currVal) || currVal-1<0 || currVal-65535>0){
							cb('Range: 1-65535');
						}else{
							cb();
						}
					}else {
						if(enable == 1) {
							cb('Range: 1-65535');
						}else {
							cb();
						}
					}
				},
				validateFileName = function(rule,value,callback) {
		        	var value = vm.fileName; 
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else if(!fileFormatMatch(value,"xlsx,xls")){
                        callback(new Error('<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>'))
                    }else {
						callback();
					} 
				},
				validWiFiEnable = function(rule,value,cb){
					if(value == '1' || value == '0') {
						cb();
					}else {
						cb('<%=rb.getString("QingXuanZe")%>');
					}
				},
				validSsid = function(rule,value,cb){
					if(vm.mForm.wifiEnable != '1') {
						cb();
					}else if(value) {
						cb();
					}else {
						cb('<%=rb.getString("ShuRuBiTianXiang")%>');
					}
				},
				validEncryption = function(rule,value,cb){
					if(vm.mForm.wifiEnable != '1') {
						cb();
					}else if(value) {
						cb();
					}else {
						cb('<%=rb.getString("QingXuanZe")%>');
					}
				},
				validPassphrase = function(rule,value,cb){
					if(vm.mForm.wifiEnable == '1' && vm.mForm.wifiEncryption != 'open') {
						if(value && value.length >= 8 && value.length <= 64) {
							cb();
						}else {
							cb('<%=rb.getString("ZiFuChang")%>: 8 - 64');
						}
					}else {
						cb();
					}
				},
				createChanel = function(chStr) {
					var vm = this,
						list = chStr.split(','),
						arr = [];
	
					list.map(function(seg){
						var range = seg.split('~');
	
						for(var i = range[0] - 0; i <= range[1] - 0; i += 4) {
							arr.push(i+'');
						}
					});
	
					return arr;
				},
				validSsidWifi5 = function(rule,value,cb){
					if(vm.mWifi5Form.wifi5WifiEnable != '1') {
						cb();
					}else if(value) {
						cb();
					}else {
						cb('<%=rb.getString("ShuRuBiTianXiang")%>');
					}
				},
				validEncryptionWifi5 = function(rule,value,cb){
					if(vm.mWifi5Form.wifi5WifiEnable != '1') {
						cb();
					}else if(value) {
						cb();
					}else {
						cb('<%=rb.getString("QingXuanZe")%>');
					}
				},
				validPassphraseWifi5 = function(rule,value,cb){
					if(vm.mWifi5Form.wifi5WifiEnable == '1' && vm.mWifi5Form.wifi5WifiEncryption != 'open') {
						if(value && value.length >= 8 && value.length <= 64) {
							cb();
						}else {
							cb('<%=rb.getString("ZiFuChang")%>: 8 - 64');
						}
					}else {
						cb();
					}
				};
            return {
            	wifi5Chanel: {
					''    : [],
					'SC1' : createChanel('36~48'),
					'SC2' : createChanel('36~64'),
					'SC3' : createChanel('52~64'),
					'SC4' : createChanel('149~161'),
					'SC5' : createChanel('149~165'),
					'SC6' : createChanel('36~48,149~161'),
					'SC7' : createChanel('36~48,149~165'),
					'SC8' : createChanel('36~64,100~140'),
					'SC9' : createChanel('36~64,149~161'),
					'SC10': createChanel('52~64,149~161'),
					'SC11': createChanel('52~64,149~165'),
					'SC12': createChanel('36~64,100~120,149~161'),
					'SC13': createChanel('36~64,100~116,132~140'),
					'SC14': createChanel('36~64,100~124,149~161'),
					'SC15': createChanel('36~64,100~140,149~161'),
					'SC16': createChanel('36~64,100~140,149~165'),
					'SC17': createChanel('52~64,100~140,149~161'),
					'SC18': createChanel('56~64,100~140,149~161'),
					'SC19': createChanel('36~64,100~116,132~140,149~165'),
					'SC20': createChanel('36~64,100~116,136~140,149~165')
				},
				addOrEditForm: {
            		policyName: '',
            		productModule: [],
            		upgradeEnable: false, //升级开关
            		configEnable: false, //参数配置开关
            		specifyVersionType: '0', 
            		targetVersion: '',
            		orginalVersion: '',
            		//参数配置
            		//Network WLAN
					wlanEnable: '',
					wifiChannel: '',
					wifiBandwidth: '', 					
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: '',
					wifi1Enable: '',
					wifi1Ssid: '',
					wifi1Encryption: '',
					wifi1Passphrase: '',				
					wifi2Enable: '',
					wifi2Ssid: '',
					wifi2Encryption: '',
					wifi2Passphrase: '',				
					wifi3Enable: '',
					wifi3Ssid: '',
					wifi3Encryption: '',
					wifi3Passphrase: '',
					//wifi5
					wifi5WlanEnable: '', //wifi5 enable
					wifi5WifiMode: '', //network mode
					wifi5WifiChannel: '',//channel
					wifi5WifiBandwidth: '',//channel bandwith
					wifi5WifiSupportChannel: '',//support channel 
					
					wifi5WifiSsid: '',
					wifi5WifiEncryption: '',
					wifi5WifiPassphrase: '',

					wifi5Wifi1Enable: '',
					wifi5Wifi1Ssid: '',
					wifi5Wifi1Encryption: '',
					wifi5Wifi1Passphrase: '',
					
					wifi5Wifi2Enable: '',
					wifi5Wifi2Ssid: '',
					wifi5Wifi2Encryption: '',
					wifi5Wifi2Passphrase: '',
					
					wifi5Wifi3Enable: '',
					wifi5Wifi3Ssid: '',
					wifi5Wifi3Encryption: '',
					wifi5Wifi3Passphrase: '',
					
					//network dmz
					dmzEnable: '',
					dmzHostAddress: '',	
					//network lan
					lanEnable: '',
					lanDNS1: '',
					lanDNS2: '',
					lanDHCPEnable: '',
					lanOption: '',
					//lte	pci lock				
					scanMode: '',
					pci: '',
					onlyEarfcn: '',
					earfcnStart: '',
					pciEnd: '',
					onlyPci: '',
					earfcnList: [],
					earfcnPciList: [],
					pciList: [],												
					//system web access
					uiPassword: '',
					httpsEnable: '',
					httpsWanEnable: '',
					wanAccEnable: '',
					rangeIp: '',
					//system ping watchdog
					watchDogEnable: '',
					watchDogPingIp: '',
					watchDogPingTimeout: '',
					watchDogPingCount: '',
					watchDogFailureReboot: '', 
				},
				addOrEditRule: {
					policyName: [{validator: validatePolicyName}],
					productModule: [{validator: validateProModule}],
					orginalVersion: [{validator: validateVersion}],
					targetVersion: [{validator: validateTarget}],
					
					watchDogPingIp:[{validator: watchdogValidateIP}],
					watchDogPingTimeout:[{validator: watchdogValid}],
					watchDogPingCount:[{validator: watchdogValid}],
					watchDogFailureReboot:[{validator: watchdogValid}]
				},
				productModels: [],
				functionModulesSelect: '0',
				parameterConfigActive: 'network',
				earfcnErrorMessage: '',
				earfcnPciErrorMessage: '',
				pciErrorMessage: '',
				earfcnTip: false,
				editBtnShow: true,
				AddBtnShow: true,
				setDefault: '1', 
				//----------------------------------software upgrade
				//version
            	tarVersionList: [],
                addVersionShow: false,
                addVersionBtnShow: true,
                resultOriginalVersionShow: false,
            	softwareAddVersionForm: {
            		versionStr: '',
					versionList: [],
					itemTest:'',           	
					orginalVersion: '',
            	},
            	softwareAddVersionRules: {},
            	versionErrorMessage: '',
            	versionMessage: '',
            	resultOriginalVersionList: [],
            	softwareOriginalVersionList: [],
				curSelectVersionData: [],
            	
				readonly: false,
                messages: {
                    placeholder: '<%=rb.getString("ChuShiBanBen") %>'
                },
                longLength: 200,
                searchText: '',                      
                ipsecKeys: [],
                enable: '0',
				searchValue: "",
                height:"100%",
                policyTitle: 'Add Policy',
                curType: 'add',
                //param config
                showImportCard: false,
                selectOptions:[   				
    				{ text: 'Disable', value: '0' },
    				{ text: 'Enable', value: '1' }
    			],
                tbData: [],
                profileListData: [],
                //ip table
                ipTableData: [],
                isLANEnable: true,
				//add ip
				addIpShow: false,
				addIpForm:{
					ipStart: '',
					ipEnd: '',
					ipGroup: []
				},
				errorMessage: '',
				//edit ip
				editIpShow: false,
				editIpForm:{
					ipStart: '',
					ipEnd: ''
				},
				editIndex: '',
				pageSize: 50,
				currentPage: 1,
				editErrorMsg: '',
				oldEditIpStart: '',
				oldEditIpEnd: '',
				notSupportShow: false,
				//end ip
				//import
				showImportCard:false,
				importRuleForm: {
		            uploadFileUrl: '',
		       	},	         	          
		        fileParams:{},              
		        fileName:'',	            					
				showFileTip:false,
				fileList:[],
				filePath:'',
				importRules: {	           		
					fileName:[
		            	{ validator: validateFileName},
		            ]                   
		        },
				wifiMDLShow: false,
				mForm: {
					id: '',
					wifiEnable: '',
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: '',
					showPassword: false
				},
				mFormRules: {
					wifiEnable: [{validator: validWiFiEnable}],
					wifiSsid: [{validator: validSsid}],
					wifiEncryption: [{validator: validEncryption}],
					wifiPassphrase: [{validator: validPassphrase}]
				},
				//wifi5
				tbDataWifi5: [],
				wifi5WifiMDLShow: false,
				mWifi5Form: {
					id: '',
					wifi5WifiEnable: '',
					wifi5WifiSsid: '',
					wifi5WifiEncryption: '',
					wifi5WifiPassphrase: '',
					wifi5ShowPassword: false
				},
				mWifi5FormRules: {
					wifi5WifiEnable: [{validator: validWiFiEnable}],
					wifi5WifiSsid: [{validator: validSsidWifi5}],
					wifi5WifiEncryption: [{validator: validEncryptionWifi5}],
					wifi5WifiPassphrase: [{validator: validPassphraseWifi5}]
				},
				wpakeys: {
					'OPEN': '',
					'WPA': 'TKIP',
					'WPA2': 'AES',
					'WPAWPA2': 'TKIP/AES'
				},
				modeKeys: {
					'OPEN': 'OPEN',
					'WPA': 'WPAPSK',
					'WPA2': 'WPA2PSK',
					'WPAWPA2': 'WPAPSK/WPA2PSK'
				},
				operateType:'',
			 	enbQuery: {
					searchText: '',
					productType: '',
					timeZone: timeZone
				},
				curPolicyId: '',
            }
        },
        computed: {
        	isWifiClosed() {
 				return this.addOrEditForm.wlanEnable != '1';
 			},
 			isWifi5Closed() {
				return this.addOrEditForm.wifi5WlanEnable != '1';
			},
        },
        watch: {           
    		curSelectVersionData(row){
    			if(row.length != 0){
    				this.versionMessage = '';
    			}    			
    		},
    		'addOrEditForm.specifyVersionType': function(val) {
                if(val == '0'){
                	this.$refs.addOrEditForm.validateField('orginalVersion');
                 }else{
                	 //1 - 开关打开
                	 this.addVersionShow = false;
                	 this.resultOriginalVersionList = []; //已添加初始版本不置灰
                	 this.$refs.addOrEditForm.clearValidate('orginalVersion');
                 }
             },
             'addOrEditForm.upgradeEnable': function(val) {
                 var vm = this;
                 if(val == true){
                	vm.$refs.addOrEditForm.validateField('orginalVersion');
                 	vm.$refs.addOrEditForm.validateField('targetVersion');
                 	vm.$refs.addOrEditForm.validateField('productModule');
                 }else{
                 	vm.$refs.addOrEditForm.clearValidate('orginalVersion');
                 	vm.$refs.addOrEditForm.clearValidate('targetVersion');
                 	vm.$refs.addOrEditForm.clearValidate('productModule');
                 	//添加初始版本弹窗关闭
                	vm.addVersionShow = false;
                 }                    
             },
             'addOrEditForm.productModule': function(val) {
            	 var vm = this;
                 if(val) {
                	 vm.addOrEditForm.productModel = val;
                 }
             },
             'addOrEditForm.targetVersion': function(val) {
                 var vm = this,
                     row = vm.tarVersionList.filter(function(item){ return item.id == val})[0];
                 try{
                     vm.softwareOriginalVersionList = [];
                 }catch(e){}
                 
             },
             ipTableData: function(tableData) {
            	 var vm = this;                 
                 if(tableData.length > 0){
                	vm.addOrEditForm.rangeIp = tableData.map((item,index) => {
						if(item.ipEnd ){
							return item.ipStart + '-' + item.ipEnd;
						}else{
							return item.ipStart;
						}								
					}).join(';');
				}else{
					vm.addOrEditForm.rangeIp = '';
				}
            },
            'addOrEditForm.lanOption': function(val) {
            	var vm = this;
    			//自动
    			if(vm.addOrEditForm.lanOption == '1' || vm.addOrEditForm.lanOption == ''){
    				vm.addOrEditForm.lanDNS1 = '';
    				vm.addOrEditForm.lanDNS2 = '';
    			}
            }
        },
        methods: {
            cpeInit(type, id, cpeOrEnb) {
                var vm = this;
                vm.curPolicyId = id;
                vm.readonly = type == 'readonly';
                if(type == 'add'){
                	vm.policyTitle = '<%=rb.getString("XinZengCeLue")%>';
                	vm.curType = 'add'
                }else if(type == 'modify'){
                	vm.policyTitle = '<%=rb.getString("XiuGaiCeLue")%>';
                	vm.curType = 'modify'
                }else{
                	vm.policyTitle = '<%=rb.getString("ChaKanCeLue")%>';
                	vm.curType = 'view'
                } 
                if(vm.readonly == true){
					vm.AddBtnShow = false;
					vm.editBtnShow = false;
				}
				if(type == 'add'){
					vm.tbData = [];
					vm.reloadTable();
					vm.tbDataWifi5 = [];
					vm.wifi5ReloadTable();
					// 初始化form原始值
					/*vm.$nextTick(function(){
						initForm(vm.$refs.addOrEditForm);					
					}); */
				}
				//产品型号
                axios.post("${ctx}/plugandplay/policy/getProductModelList.action").then(function(res){
                	var rows = res.data||[];
					vm.productModels = rows;
					if(vm.readonly || type == 'modify') {
						//vm.tbData = [];
   	                    vm.getInfoById(id,cpeOrEnb);
   					}
				}).catch(function(){});
                //目标版本
            },
           //根据已选产品类型加载对应的目标版本
            commonTargetVerList(){
            	var vm = this;
            	
            	axios.post("${ctx}/plugandplay/policy/getCpeTargetVersionList.action", stringify({
                	modelName: vm.addOrEditForm.productModule.join(',')
                })).then(function(res){
                   	var rows = res.data||[];
   					vm.tarVersionList = rows;
   				}).catch(function(){});
            },
          	//根据已选产品类型加载对应的初始版本数据
			commonOriginalVerList(){
				var vm = this;
				// 根据产品类型回显对应初始版本
                axios.post("${ctx}/plugandplay/policy/getOriginalVersionPageList.action",stringify({
                	product: vm.addOrEditForm.productType,
                    searchText: '',
                    page: 1,
                    rows: 50
                })).then(function(res){
					var data = res.data;
					vm.softwareOriginalVersionList = data.rows;
				});
			},
            getInfoById(id,cpeOrEnb) {
                var vm = this,
                    params = {
                        policyId: id,
                        type: 'CPE'
                    },
                    url = '${ctx}/plugandplay/policy/getAutoPolicyInfo.action';

                // 获取策略信息
                axios.post(url, stringify(params)).then(function(res){
					var data = res.data;
					Object.assign(vm.addOrEditForm, data);
                    //回显产品类型
					vm.addOrEditForm.productModule = (data.productType || '').split(',');
					//回显模板版本的下拉框数据
                    vm.commonTargetVerList();
					var tarverVal = vm.addOrEditForm.targetVersion,
                    	row = vm.tarVersionList.filter(function(item){ return item.id == tarverVal})[0];
                    
					//回显初始版本
					vm.addVersionBtnShow = false;
	                vm.resultOriginalVersionShow = true;
	                vm.resultOriginalVersionList = (data.originalVersionList||[]).map(function(item){
                        return {orginalVersion: item};
                    });
	              	//any
	                if(data.specifyVersionType != null){
	                	if(data.specifyVersionType == 'allExecute'){
	                		vm.addOrEditForm.specifyVersionType = '1';
		               	}else{
		               		vm.addOrEditForm.specifyVersionType = '0';               
	                	}
	                }

	               	if(data){
						var curParams = data.paramConfig;
						vm.tbData = [];
						vm.tbDataWifi5 = [];
						if(data.paramConfig  === '' || data.paramConfig  === null || data.paramConfig === undefined){
							vm.reloadTable();
							vm.wifi5ReloadTable();
						}else{
							for(var key in curParams){								
								if(key == 'wifi'){
									//vm.tbData = [];
									if(curParams[key] === null || curParams[key] === '' || curParams[key] === undefined){
										//wifi 返回值为null,并不是对象
										vm.reloadTable();
										
									}else{
										if(curParams[key] && curParams[key]['wlanEnable']){
											vm.raloadTableData( curParams[key]);
										}else{
											vm.reloadTable();
											//有 wifi, 但 enable 为 null
										}
									}
									//vm.reloadInfoForm(curParams);
								}else{
									//vm.reloadInfoForm(curParams);
								}
								if(key == 'wifi5'){
									vm.tbDataWifi5 = [];
									if(curParams[key] === null || curParams[key] === '' || curParams[key] === undefined){
										vm.wifi5ReloadTable();
									}else{
										if(curParams[key] && curParams[key]['wifi5WlanEnable']){
											vm.wifi5RaloadTableData( curParams[key]);
										}else{
											vm.wifi5ReloadTable();
										}
									}
								}
								vm.reloadInfoForm(curParams);
							} 							
						} 
					}
				});
            },	
        	//software upgrade,parameter config 三个模块切换时，使其打开的右侧内容关闭； 需校验内容是否发生变化
        	selectMethodChange(val){
        		var vm = this;
        		vm.addVersionShow = false;
        		vm.showImportCard = false;
        		vm.wifiMDLShow = false;
        		vm.wifi5WifiMDLShow = false;
        		vm.addIpShow = false;
        		vm.editIpShow = false;
        	},
        	//parameter config 中的 parameter data pool, plan for the specified device, region deploying 三个模块切换时，使其打开的右侧内容关闭； 需校验内容是否发生变化
        	clickTab(){
        		var vm = this;
        		vm.addVersionShow = false;
        		vm.showImportCard = false;
        		vm.wifiMDLShow = false;
        		vm.wifi5WifiMDLShow = false;
        		vm.addIpShow = false;
        		vm.editIpShow = false;
			},
        	//--------------------------------------------software upgrade
        	//新增版本按钮点击事件
			addVersionClick(){
				var vm = this;
				vm.addVersionShow = true;
				vm.curSelectVersionData = [];
				vm.softwareAddVersionForm.versionList = [];
				vm.$refs.softwareOriginalVersionTable.clearSelection();
				
				vm.commonOriginalVerList();
			},
			 //根据已选产品类型加载对应的初始版本数据
			commonOriginalVerList(){
				var vm = this;
				// 获取初始版本列表
                axios.post("${ctx}/plugandplay/policy/queryCpeVersionsPageList.action",stringify({
                	productModel: vm.addOrEditForm.productModule,
                    searchText: '',
                    page: 1,
                    rows: 50
                })).then(function(res){
					var data = res.data;
					vm.softwareOriginalVersionList = data.rows;
				});
			},
			//手动添加 version
			addVersionBtn(){
				var vm = this, versionText = vm.softwareAddVersionForm.versionStr, str = '';
				if(versionText == ''){
					vm.versionErrorMessage = '<%=rb.getString("ShuRuBanBenHao")%>';
				}else{
					if(versionText){
						str = versionText;
						if(vm.softwareAddVersionForm.versionList.indexOf(str) == -1){
							vm.softwareAddVersionForm.versionList.push(str);
							vm.softwareAddVersionForm.versionStr = '';
							vm.versionErrorMessage = '';
							vm.versionMessage = '';
							vm.$refs.softwareAddVersionForm.validateField('itemTest');
						}else{
							vm.versionErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}else{
						vm.versionErrorMessage = '<%=rb.getString("ShuRuBanBenHao")%>';
					}
				}
			},
			//删除version
			removeVersion(item){
				var vm = this;
				var index = vm.softwareAddVersionForm.versionList.indexOf(item);
				if(index !== -1){
					vm.softwareAddVersionForm.versionList.splice(index,1)
				}
				vm.versionErrorMessage = '';
			},
			/**
			* 列表选中
			* @param selection{Array}   选中数据
			*/
			versionBatchSelect(selection){
				var vm = this; 
				vm.curSelectVersionData = selection;
			},
        	//add version submit
        	addVersionSubmit(){
        		var vm = this;
        		var manualVersionList = vm.softwareAddVersionForm.versionList;
        		var versionList = vm.curSelectVersionData.map(function(item){ return item.software_version});
        		//手动添加版本号（还未在OMC上线或上传的版本号）和已上线OMC 的版本号二选其一皆可
        		if(manualVersionList.length == 0 && vm.curSelectVersionData.length == 0){
        			vm.versionMessage = '<%=rb.getString("ZhiShaoXuanZeYiZhongBanBenFangShi")%>';
        			return;
        		}else{        		
        			vm.versionMessage = '';
        			//数组合并 去重
            		var mergeVersionList = [];
            		var mergelist = manualVersionList.concat(versionList);
            		for(var i =0, len = mergelist.length; i<len; i++){
            			if(mergeVersionList.indexOf(mergelist[i]) === -1){
            				mergeVersionList.push(mergelist[i])
            			}
            		}
            		var curList = mergeVersionList.map(function(item){return {orginalVersion: item }});
            		if(vm.resultOriginalVersionList.length > 0){
            			//多次添加时，需将新增及已有的版本号合并去重
            			var length1 = vm.resultOriginalVersionList.length;
            			var length2 = curList.length;
            			for (var i = 0; i<length1; i++){
            				for(var j= 0; j<length2; j++){
            					if(vm.resultOriginalVersionList.length > 0){
            						if(vm.resultOriginalVersionList[i]['orginalVersion'] === curList[j]['orginalVersion']){
            							vm.resultOriginalVersionList.splice(i,1);
            							length1 --;
            						}
            					}
            				}
            			}
            			for(var n = 0; n< curList.length; n++){
            				vm.resultOriginalVersionList.push(curList[n]);
            			}
            		}else{
            			vm.resultOriginalVersionList = mergeVersionList.map(function(item){return {orginalVersion: item }})
            		}
            		
            		vm.addOrEditForm.orginalVersion = vm.resultOriginalVersionList.map(function(item){ return item.orginalVersion;}).join(',');
            		vm.addVersionShow = false;
            		vm.addVersionBtnShow = false;
                    vm.resultOriginalVersionShow = true;
        		}        		
        	},

        	//add version cancel
        	addVersionCancel(){
        		var vm = this;
        		vm.addVersionShow = false;
        		vm.curSelectVersionData = [];
        		vm.$refs.softwareAddVersionForm.resetFields();
        		vm.softwareAddVersionForm.versionList = [];
        		vm.$refs.softwareOriginalVersionTable.clearSelection();
        	},
        	//从结果版本列表中删除
        	deleteVersionItem(item){
				var vm = this;
				var index = vm.resultOriginalVersionList.indexOf(item);
				if(index !== -1){
					vm.resultOriginalVersionList.splice(index,1)
				}
			},
			//clear 操作
			clearVersionBtnClick(){
				this.resultOriginalVersionList = [];
			},

			//-------------------------------------参数配置

			// 打开导入弹出框
			importFileBtn(){
				var vm = this;
				vm.showImportCard = true;
				vm.wifiMDLShow = false;
				vm.wifi5WifiMDLShow = false;
				vm.addIpShow = false;
				vm.editIpShow = false;
			},
			//导入
			checkFile(res,file){  
				var vm = this;
				if(res.success){
					if(res.suc_count>0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}				
					vm.closeFileSelect();
					vm.showImportCard = false;
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.upload.uploadFiles;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
	    	
			/**
			* 选择文件后，校验格式，并赋值页面显示 
			* @param file{object}   文件信息
			* @param fileList{Array}  文件列表
			*/ 
			fileChange(file,fileList){ 
				var vm = this;
				vm.fileName = file.name;
				vm.fileParams.FileName = file.name;
			},
			
			// 选择文件
			fileSelect(){  
				var vm =this;
				vm.$refs.upload.clearFiles();
				vm.$refs['file_up'].click();
			},
			
			// 移除导入文件
			closeFileSelect(){
				var vm = this;
				vm.fileName = '';			
				vm.$refs.upload.clearFiles();
			},
						
			/**
			* 文件上传之前
			* @param file{object}   文件信息
			*/ 
			beforeUpload(file){
				var vm = this, importUrl, fileName = file.name,
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};

				fd.append('uploadFile',file); //文件流
				axios.post('${ctx}/cpe/batchconfig/importBatchConfigurationInfos.action',fd,config).then(function(res){
					var data = res.data;
					if(data["success"]){	
						vm.$message.success('<%=rb.getString("ChengGong")%>');						
						var curParams = data.paramConfig;
						vm.tbData = [];
						vm.tbDataWifi5 = [];
						if(vm.curType == 'modify'){
		            		var curPolicyId = vm.curPolicyId;
		            	}else{
		            		var curPolicyId = '';
		            	}
						var resetBefore = {
							policyId: curPolicyId,
							policyName: vm.addOrEditForm.policyName,
							productModule: vm.addOrEditForm.productModule,
							targetVersion: vm.addOrEditForm.targetVersion,
							specifyVersionType: vm.addOrEditForm.specifyVersionType,
							upgradeEnable: vm.addOrEditForm.upgradeEnable,
							configEnable: vm.addOrEditForm.configEnable
						}
						vm.$refs.addOrEditForm.resetFields(); //表单置空重新渲染,所有字段都被重置（x）
						//其它模块数据重新赋值
						Object.assign(vm.addOrEditForm, resetBefore);

						vm.closeImportParams();
						if(data.paramConfig  === '' || data.paramConfig  === null || data.paramConfig === undefined){
							vm.reloadTable();
							vm.wifi5ReloadTable();
						}else{
							for(var key in curParams){
								vm.ipTableData = [];
								if(key == 'wifi'){
									vm.tbData = [];
									if(curParams[key] === null || curParams[key] === '' || curParams[key] === undefined){
										//wifi 返回值为null,并不是对象
										vm.reloadTable();
										
									}else{
										if(curParams[key] && curParams[key]['wlanEnable']){
											vm.raloadTableData( curParams[key]);
										}else{
											vm.reloadTable();
											//有 wifi, 但 enable 为 null
										}
									}
									//vm.reloadInfoForm(curParams);
								}else{
									//vm.reloadInfoForm(curParams);
								}
								if(key == 'wifi5'){
									vm.tbDataWifi5 = [];
									if(curParams[key] === null || curParams[key] === '' || curParams[key] === undefined){
										vm.wifi5ReloadTable();
									}else{
										if(curParams[key] && curParams[key]['wifi5WlanEnable']){
											vm.wifi5RaloadTableData( curParams[key]);
										}else{
											vm.wifi5ReloadTable();
										}
									}
								}
								vm.reloadInfoForm(curParams);
							}
						}
					}else{
						vm.$message.error(data["msg"])
	        			vm.fileList = [];
						vm.fileName = '';
						vm.$refs.importRuleForm.resetFields();
					} 
				})			
				return false;
			},
			
			reloadInfoForm(curParams){
				var vm = this;
				if(curParams != null || curParams != undefined){
					for(var key in curParams){
						if(typeof curParams[key] === 'object'){							
							vm.reloadInfoForm(curParams[key])							
						}else{
							vm.addOrEditForm[key] = curParams[key];
							if(key == 'pci'){
								if( curParams[key] === '' || curParams[key] === null || curParams[key] === undefined){
									//全部为空数组
								}else{
									var newEarfcnPci = curParams[key].split(';');
									if(vm.addOrEditForm.scanMode == 'freqpreferred'){
										vm.addOrEditForm.earfcnList = newEarfcnPci.map(function(item){
											return item
										});
									}else if(vm.addOrEditForm.scanMode == 'pcilock'){
										vm.addOrEditForm.earfcnPciList = newEarfcnPci.map(function(item){
											var getRowData = {};
											if(item.includes(',')){
												getRowData = {
													startValue : item.split(',')[0],
													endValue : item.split(',')[1]
						      		    		}
											}
											return getRowData
										});
									}else if (vm.addOrEditForm.scanMode == 'pcionlylock'){
										vm.addOrEditForm.pciList = newEarfcnPci.map(function(item){
											return item
										});
									}
								}
							}
							if(key == 'rangeIp'){
								if(curParams[key] === '' || curParams[key] === null || curParams[key] === undefined){
									vm.ipTableData = [];
								}else{
									var newIpListData = curParams[key].split(';');
									if(newIpListData != '' || newIpListData.length > 0){
										vm.ipTableData = newIpListData.map((item,index) => {
											if(item.includes('-')){
												return {
													ipStart : item.split('-')[0],
						      						ipEnd : item.split('-')[1]
						      		    		}
											}else{
												return {
													ipStart : item
							      		    	}
											}
										});	
									}
								}
							}								
						}
					}
				} 
				vm.$nextTick(function(){
					initForm(vm.$refs.addOrEditForm);					
				}); 
			},
		
			reloadTable(){
				var vm = this;
				vm.tbData.push({
					id: 'main',
					wifiEnable: '',
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: ''
				}); 
				// networkForm更新
				Object.assign(vm.addOrEditForm,{
					wifiSsid: '',
					wifiEncryption: '',
					wifiPassphrase: ''
				});
				// sub 
				 ['1','2','3'].map(function(item){
					var row = {},pre = 'wifi',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase';							
					
					row['id'] = item;
					row['wifiEnable'] = '';
					row['wifiSsid'] = '';
					row['wifiEncryption'] = '';
					row['wifiPassphrase'] = '';

					// networkForm更新
					vm.addOrEditForm[enableKey] = '';
					vm.addOrEditForm[idKey] = '';
					vm.addOrEditForm[encryKey] = '';
					vm.addOrEditForm[phraKey] = '';
					vm.tbData.push(row); 
				}); 
			},
			raloadTableData( data){
				var vm = this;
				vm.tbData.push({
					id: 'main',
					wifiEnable: data["wlanEnable"],
					wifiSsid: data["wifiSsid"],
					wifiEncryption: data["wifiEncryption"],
					wifiPassphrase: data["wifiPassphrase"],
				});
				Object.assign(vm.addOrEditForm,{
					wifiSsid: data["wifiSsid"],
					wifiEncryption: data["wifiEncryption"],
					wifiPassphrase: data["wifiPassphrase"],
				}); 
				// sub 
				['1','2','3'].map(function(item){
					var row = {},pre = 'wifi',suf = 'Old',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',						
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase';
					row['id'] = item;
					row['wifiEnable'] = data[enableKey];
					row['wifiSsid'] = data[idKey];
					row['wifiEncryption'] = data[encryKey];
					row['wifiPassphrase'] = data[phraKey];
					vm.addOrEditForm[enableKey] = data[enableKey];
					vm.addOrEditForm[idKey] = data[idKey];
					vm.addOrEditForm[encryKey] = data[encryKey];
					vm.addOrEditForm[phraKey] = data[phraKey];
					vm.tbData.push(row);
				}); 
			},
			//wifi5
			wifi5ReloadTable(){
				var vm = this;
				vm.tbDataWifi5.push({
					id: 'main',
					wifi5WifiEnable: '',
					wifi5WifiSsid: '',
					wifi5WifiEncryption: '',
					wifi5WifiPassphrase: ''
				}); 
				// Form更新
				Object.assign(vm.addOrEditForm,{
					wifi5WifiSsid: '',
					wifi5WifiEncryption: '',
					wifi5WifiPassphrase: ''
				});
				// sub 
				 ['1','2','3'].map(function(item){
					var row = {},pre = 'wifi5Wifi',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase';							
					
					row['id'] = item;
					row['wifi5WifiEnable'] = '';
					row['wifi5WifiSsid'] = '';
					row['wifi5WifiEncryption'] = '';
					row['wifi5WifiPassphrase'] = '';

					// Form更新
					vm.addOrEditForm[enableKey] = '';
					vm.addOrEditForm[idKey] = '';
					vm.addOrEditForm[encryKey] = '';
					vm.addOrEditForm[phraKey] = '';
					vm.tbDataWifi5.push(row); 
				}); 
			},
			wifi5RaloadTableData( data){
				var vm = this;
				vm.tbDataWifi5.push({
					id: 'main',
					wifi5WifiEnable: data["wifi5WlanEnable"],
					wifi5WifiSsid: data["wifi5WifiSsid"],
					wifi5WifiEncryption: data["wifi5WifiEncryption"],
					wifi5WifiPassphrase: data["wifi5WifiPassphrase"]
				});
				// Form更新
				Object.assign(vm.addOrEditForm,{
					wifi5WifiSsid: data["wifi5WifiSsid"],
					wifi5WifiEncryption: data["wifi5WifiEncryption"],
					wifi5WifiPassphrase: data["wifi5WifiPassphrase"],
				}); 
				// sub 
				['1','2','3'].map(function(item){
					var row = {},pre = 'wifi5Wifi',suf = 'Old',
						enableKey = pre+item+'Enable',
						idKey = pre+item+'Ssid',						
						encryKey = pre+item+'Encryption',
						phraKey = pre+item+'Passphrase';
					row['id'] = item;
					row['wifi5WifiEnable'] = data[enableKey];
					row['wifi5WifiSsid'] = data[idKey];
					row['wifi5WifiEncryption'] = data[encryKey];
					row['wifi5WifiPassphrase'] = data[phraKey];
					// Form更新
					vm.addOrEditForm[enableKey] = data[enableKey];
					vm.addOrEditForm[idKey] = data[idKey];
					vm.addOrEditForm[encryKey] = data[encryKey];
					vm.addOrEditForm[phraKey] = data[phraKey];
					vm.tbDataWifi5.push(row);
				}); 
			},		
			/*确定导入*/
	        uploadParams() {
				var vm = this;
				vm.$refs.importRuleForm.validate((valid) => {
	                if (valid) {
	                	vm.$refs.upload.submit();                   	
	                }
	            }) 				
			},
			
			// 关闭导入弹出框
			closeImportParams(){
				var vm = this;			
				vm.fileList = [];
				vm.fileName = '';
				vm.$refs.importRuleForm.resetFields();
				vm.showImportCard = false;
			},
			//下载模板
			exportTemplate(){
				exportByForm('${ctx}/cpe/batchconfig/downloadTemplate.action', {});
			},
			
			//导出
			exportFileBtn(){
				var vm = this, params = {}, curRangeIp = '', curPciParams = '', curForm = vm.addOrEditForm, scanMode = vm.addOrEditForm.scanMode;
				//earfcn
				if(scanMode == 'freqpreferred') {
					curPciParams = curForm.earfcnList.map(function(item){
						return item;
					}).join(';');
				}
				// earfcn and pci
				if(scanMode == 'pcilock') {
					curPciParams = curForm.earfcnPciList.map(function(item){
						var curPci = item.startValue +','+ item.endValue;
						return curPci;
					}).join(';');
				}
				//pci
				if(scanMode == 'pcionlylock') {
					curPciParams = curForm.pciList.map(function(item){
						return item;
					}).join(';');
				}
				vm.tbData.map(function(row){
					if(['1','2','3'].includes(row.id)) {
						var index = row.id,
							pre = 'wifi',
							enableKey = pre+index+'Enable',
							idKey = pre+index+'Ssid',
							encryKey = pre+index+'Encryption',
							phraKey = pre+index+'Passphrase';
						curForm[enableKey] = row['wifiEnable'];
						curForm[idKey] = row['wifiSsid'];
						curForm[encryKey] = row['wifiEncryption'];
						curForm[phraKey] = row['wifiPassphrase'];
					}else {
						Object.assign(curForm,{
							wifiSsid: row['wifiSsid'],
							wifiEncryption: row['wifiEncryption'],
							wifiPassphrase: row['wifiPassphrase']
						});
					}
				});
				vm.tbDataWifi5.map(function(row){
					if(['1','2','3'].includes(row.id)) {
						var index = row.id,
							pre = 'wifi5Wifi',
							enableKey = pre+index+'Enable',
							idKey = pre+index+'Ssid',
							encryKey = pre+index+'Encryption',
							phraKey = pre+index+'Passphrase';
						curForm[enableKey] = row['wifi5WifiEnable'];
						curForm[idKey] = row['wifi5WifiSsid'];
						curForm[encryKey] = row['wifi5WifiEncryption'];
						curForm[phraKey] = row['wifi5WifiPassphrase'];
					}else {
						Object.assign(curForm,{
							wifi5WifiSsid: row['wifi5WifiSsid'],
							wifi5WifiEncryption: row['wifi5WifiEncryption'],
							wifi5WifiPassphrase: row['wifi5WifiPassphrase']
						});
					}
				});

				var paramConfig= {
					wifi:{
						wlanEnable : vm.addOrEditForm.wlanEnable,
						wifiChannel: vm.addOrEditForm.wifiChannel,
						wifiBandwidth : vm.addOrEditForm.wifiBandwidth,
						
						wifiSsid: vm.addOrEditForm.wifiSsid,
						wifiEncryption : vm.addOrEditForm.wifiEncryption,
						wifiPassphrase: vm.addOrEditForm.wifiPassphrase,
						
						wifi1Enable : vm.addOrEditForm.wifi1Enable,
						wifi1Ssid: vm.addOrEditForm.wifi1Ssid,
						wifi1Encryption: vm.addOrEditForm.wifi1Encryption,
						wifi1Passphrase : vm.addOrEditForm.wifi1Passphrase,
						
						wifi2Enable: vm.addOrEditForm.wifi2Enable,
						wifi2Ssid : vm.addOrEditForm.wifi2Ssid,
						wifi2Encryption: vm.addOrEditForm.wifi2Encryption,
						wifi2Passphrase: vm.addOrEditForm.wifi2Passphrase,
						
						wifi3Enable: vm.addOrEditForm.wifi3Enable,
						wifi3Ssid : vm.addOrEditForm.wifi3Ssid,
						wifi3Encryption: vm.addOrEditForm.wifi3Encryption,
						wifi3Passphrase: vm.addOrEditForm.wifi3Passphrase
					},
					wifi5:{
						wifi5WlanEnable : vm.addOrEditForm.wifi5WlanEnable,
						wifi5WifiMode: vm.addOrEditForm.wifi5WifiMode,
						wifi5WifiChannel : vm.addOrEditForm.wifi5WifiChannel,
						wifi5WifiBandwidth : vm.addOrEditForm.wifi5WifiBandwidth,
						wifi5WifiSupportChannel : vm.addOrEditForm.wifi5WifiSupportChannel,
						
						wifi5WifiSsid: vm.addOrEditForm.wifi5WifiSsid,
						wifi5WifiEncryption : vm.addOrEditForm.wifi5WifiEncryption,
						wifi5WifiPassphrase: vm.addOrEditForm.wifi5WifiPassphrase,
						
						wifi5Wifi1Enable : vm.addOrEditForm.wifi5Wifi1Enable,
						wifi5Wifi1Ssid: vm.addOrEditForm.wifi5Wifi1Ssid,
						wifi5Wifi1Encryption: vm.addOrEditForm.wifi5Wifi1Encryption,
						wifi5Wifi1Passphrase : vm.addOrEditForm.wifi5Wifi1Passphrase,
						
						wifi5Wifi2Enable: vm.addOrEditForm.wifi5Wifi2Enable,
						wifi5Wifi2Ssid : vm.addOrEditForm.wifi5Wifi2Ssid,
						wifi5Wifi2Encryption: vm.addOrEditForm.wifi5Wifi2Encryption,
						wifi5Wifi2Passphrase: vm.addOrEditForm.wifi5Wifi2Passphrase,
						
						wifi5Wifi3Enable: vm.addOrEditForm.wifi5Wifi3Enable,
						wifi5Wifi3Ssid : vm.addOrEditForm.wifi5Wifi3Ssid,
						wifi5Wifi3Encryption: vm.addOrEditForm.wifi5Wifi3Encryption,
						wifi5Wifi3Passphrase: vm.addOrEditForm.wifi5Wifi3Passphrase
					},
					dmz:{
						dmzEnable : vm.addOrEditForm.dmzEnable,
						dmzHostAddress: vm.addOrEditForm.dmzHostAddress
					},
					lan:{
						lanEnable : vm.addOrEditForm.lanEnable,
						lanDHCPEnable: vm.addOrEditForm.lanDHCPEnable,
						lanDNS1 : vm.addOrEditForm.lanDNS1,
						lanDNS2: vm.addOrEditForm.lanDNS2,
						lanOption: vm.addOrEditForm.lanOption
					},
					pciLock:{
						scanMode: vm.addOrEditForm.scanMode,
						pci: curPciParams
					},
					wanAcc:{
						uiPassword : vm.addOrEditForm.uiPassword,
						httpsEnable: vm.addOrEditForm.httpsEnable,
						httpsWanEnable : vm.addOrEditForm.httpsWanEnable,
						wanAccEnable: vm.addOrEditForm.wanAccEnable,
						rangeIp: vm.addOrEditForm.rangeIp
					},
					watchDog:{
						watchDogEnable : vm.addOrEditForm.watchDogEnable,
						watchDogPingIp: vm.addOrEditForm.watchDogPingIp,
						watchDogPingTimeout : vm.addOrEditForm.watchDogPingTimeout,
						watchDogPingCount: vm.addOrEditForm.watchDogPingCount,
						watchDogFailureReboot: vm.addOrEditForm.watchDogFailureReboot
					}
				}
				params.paramConfig = JSON.stringify(paramConfig);
				exportByForm("${ctx}/cpe/batchconfig/exportBatchConfigurationInfos.action",params);
			},
			// 导入 导出 end
			clickLanOption(e){
				var vm = this;
				if(e.target.tagName === 'INPUT') return;
				
				vm.addOrEditForm.lanOption = !vm.addOrEditForm.lanOption;

				if(vm.addOrEditForm.lanOption == false){
					vm.addOrEditForm.lanOption = '';
				}
			},		
			//add  earfcn
			addEarfcnBtn(){
				var vm = this , value = vm.addOrEditForm.onlyEarfcn;
				/* if(vm.addOrEditForm.configEnable == '0'){
					vm.earfcnErrorMessage = '';
				}else{ */
					if(value && isNaN(value)) {
						vm.earfcnErrorMessage = '<%=rb.getString("PinDianGeShiCuoWu")%>';
					}else if(value) {
						if(value - 0 < 0 || value - 65535 > 0) {
							vm.earfcnErrorMessage = '<%=rb.getString("PinDianChaoChuFanWei")%>';
						}else {
							if(vm.addOrEditForm.earfcnList.indexOf(value) == -1){
								vm.addOrEditForm.earfcnList.push(value);
								vm.addOrEditForm.onlyEarfcn = '';
								vm.earfcnErrorMessage = '';
								vm.addOrEditForm.pci = vm.addOrEditForm.earfcnList.map(function(item){
									return item;							
								}).join(';');
							}else{
								vm.earfcnErrorMessage = '<%=rb.getString("YiCunZai")%>';
							}
						}
					}else {
						vm.earfcnErrorMessage = '<%=rb.getString("PinDianWeiKong")%>';
					}
				//}
			},
			//新建 earfcn and pci
			addPcilockBtn(){
				var vm = this , startValue = vm.addOrEditForm.earfcnStart, endValue = vm.addOrEditForm.pciEnd, regNum = /^\d+$/;
				/* if(vm.addOrEditForm.configEnable == '0'){
					vm.earfcnPciErrorMessage = '';
				}else{ */
					if(startValue == '' || endValue == ''){
						vm.earfcnPciErrorMessage = '<%=rb.getString("QingShuRu")%><%=rb.getString("ENBPinDian")%> , <%=rb.getString("PCI2")%>';
					}else{
						if(!regNum.test(startValue) || ( startValue - 0 < 0 || startValue - 65535 > 0)) {
							vm.earfcnPciErrorMessage = '<%=rb.getString("ENBPinDian")%> <%=rb.getString("FanWei")%>：0 ~65535';
						}else if(!regNum.test(endValue) || (endValue - 0 < 0 || endValue-503 > 0)){
							vm.earfcnPciErrorMessage = 'PCI <%=rb.getString("FanWei")%>：0~503';
						}else{
							str = startValue + "," + endValue;
							if(vm.addOrEditForm.earfcnPciList.indexOf(str) == -1){
								vm.addOrEditForm.earfcnPciList.push({startValue,endValue});
								vm.addOrEditForm.earfcnStart = '';
								vm.addOrEditForm.pciEnd = '';
								vm.earfcnPciErrorMessage = '';
								vm.addOrEditForm.pci = vm.addOrEditForm.earfcnPciList.map(function(item){
									var curPci = item.startValue +','+ item.endValue;						
									return curPci;
								}).join(';');
							}else{
								vm.earfcnPciErrorMessage = '<%=rb.getString("IPFanWeiYiCunZai")%>';
							} 
						}					
					}
				//}							
			},
			// add pci
			addPciBtn(){
				var vm = this , value = vm.addOrEditForm.onlyPci;
				/* if(vm.addOrEditForm.configEnable == '0'){
					vm.pciErrorMessage = '';
				}else{ */
					if(value && isNaN(value)) {
						vm.pciErrorMessage = '<%=rb.getString("PCIGeShiCuoWu")%>';
					}else if(value) {
						if(value-0 < 0 || value-503 > 0) {
							vm.pciErrorMessage = '<%=rb.getString("PCIChaoChuFanWei")%>';
						}else {
							if(vm.addOrEditForm.pciList.indexOf(value) == -1){
								vm.addOrEditForm.pciList.push(value);
								vm.addOrEditForm.onlyPci = '';
								vm.pciErrorMessage = '';
								vm.addOrEditForm.pci = vm.addOrEditForm.pciList.map(function(item){
									return item;
								}).join(';');
							}else{
								vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
							}
						}
					}else {
						vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
					}
				//}
			},
			removeEarfcn(item){
				var vm = this, index = vm.addOrEditForm.earfcnList.indexOf(item);
			
				if(index !== -1){
					vm.addOrEditForm.earfcnList.splice(index,1)
				}
				vm.addOrEditForm.pci = vm.addOrEditForm.earfcnList.map(function(item){
					return item;							
				}).join(';');
				vm.earfcnErrorMessage = '';
			},
			removeEarfcnPci(item){
				var vm = this, index = vm.addOrEditForm.earfcnPciList.indexOf(item);
				if(index !== -1){
					vm.addOrEditForm.earfcnPciList.splice(index,1)
				}
				vm.addOrEditForm.pci = vm.addOrEditForm.earfcnPciList.map(function(item){
					var curPci = item.startValue +','+ item.endValue;						
					return curPci;
				}).join(';');
				vm.earfcnPciErrorMessage = '';
			},
			removePci(item){
				var vm = this, index = vm.addOrEditForm.pciList.indexOf(item);
				if(index !== -1){
					vm.addOrEditForm.pciList.splice(index,1)
				}
				vm.addOrEditForm.pci = vm.addOrEditForm.pciList.map(function(item){
					return item;
				}).join(';');
				vm.pciErrorMessage = '';
			},
			
			modifyWifi(row) {
				var vm = this;
				vm.wifiMDLShow = true;
				vm.wifi5WifiMDLShow = false;
				vm.showImportCard = false;
				vm.$nextTick(function(){
					vm.$refs.modifyForm.resetFields();
					Object.assign(vm.mForm, row);
				});
			},
			saveMbssidModify() {
				var vm = this;
				
				vm.$refs.modifyForm.validate(function(valid){
					if(valid) {
						vm.tbData.map(function(item){
							if(item.id == vm.mForm.id) {
								if(vm.mForm.wifiEnable == '0'){
									//置空表格SSID 和 security Mode
									vm.mForm.wifiSsid = '';
									vm.mForm.wifiEncryption = '';
									vm.mForm.wifiPassphrase = '';
									vm.mForm.showPassword = false
								}
								Object.assign(item, vm.mForm);
							}
						});
						vm.wifiMDLShow = false;
					}
				});
			},
			cancelMbssidModify(){
				var vm = this;
				vm.wifiMDLShow = false;
			},
			//wifi5
			modifyWifi5(row) {
				var vm = this;

				vm.wifi5WifiMDLShow = true;
				vm.wifiMDLShow = false;
				vm.showImportCard = false;
				vm.$nextTick(function(){
					vm.$refs.modifyWifi5Form.resetFields();
					Object.assign(vm.mWifi5Form, row);
				});
			},
			saveModifyWifi5() {
				var vm = this;
				vm.$refs.modifyWifi5Form.validate(function(valid, errors){
					if(valid) {
						vm.tbDataWifi5.map(function(item){
							if(item.id == vm.mWifi5Form.id) {
								if(vm.mWifi5Form.wifi5WifiEnable == '0'){
									//置空表格SSID 和 security Mode
									vm.mWifi5Form.wifi5WifiSsid = '';
									vm.mWifi5Form.wifi5WifiEncryption = '';
									vm.mWifi5Form.wifi5WifiPassphrase = '';
									vm.mWifi5Form.wifi5ShowPassword = false;
								}
								Object.assign(item, vm.mWifi5Form);
							}
						});
						vm.wifi5WifiMDLShow = false;
					}
				});
			},
			cancelWifiMbssidModify(){
				var vm = this;
				vm.wifi5WifiMDLShow = false;
			},
        	handleSizeChange(val){
				this.pageSize = val;
			},
			handleCurrentChange(val){
				this.currentPage = val;
			},
			//new IP
        	systemAddIp(){
				var vm = this;
				vm.addIpShow = true;
				vm.editIpShow = false;
				vm.showImportCard = false;
			},
			//update ip
			updateIp(row,index){ 
				var vm = this;
				vm.editIndex = index;
				vm.editIpForm.ipStart = row.ipStart;
				vm.editIpForm.ipEnd = row.ipEnd;
				vm.oldEditIpStart = row.ipStart,
				vm.oldEditIpEnd = row.ipEnd;
				vm.editIpShow = true;
				vm.addIpShow = false;
				vm.showImportCard = false;
		    },
		    saveEditIp(){
		    	var vm = this,
					reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
					ipStart = vm.editIpForm.ipStart,
					ipEnd = vm.editIpForm.ipEnd;
		    	
				//endIp不是必填项
				if(ipStart === '' || ipStart === null || ipStart === undefined){					
					vm.editErrorMsg = '<%=rb.getString("IPDiZhiBuNengWeiKong")%>';
				}else{
					//如果只有start IP
					 if(ipEnd == '' || ipEnd == null){
						if(reg.test(ipStart)){
							str = ipStart;
							if(ipStart == vm.oldEditIpStart){
								vm.editErrorMsg = '<%=rb.getString("IPYiCunZai")%>';
							}else{
								vm.ipTableData.splice(vm.editIndex,1,vm.editIpForm)
								vm.closeEditIp();
							}
						}else{
							vm.editErrorMsg = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}else{
						//如果star ip ，end ip都有
						if(reg.test(ipStart) && reg.test(ipEnd) && vm.compareIp(ipStart,ipEnd)){
							if(ipStart == vm.oldEditIpStart && ipEnd == vm.oldEditIpEnd){
								vm.editErrorMsg = '<%=rb.getString("IPFanWeiYiCunZai")%>';
							}else{
								vm.ipTableData.splice(vm.editIndex,1,vm.editIpForm)
								vm.closeEditIp();
							}
						}else{
							vm.editErrorMsg = '<%=rb.getString("IPDiZhiFeiFa")%>';
						 }
					 }	
				} 
		    },
		    closeEditIp(){
		    	var vm = this;
		    	vm.editIpForm = {
					ipStart:'',
					ipEnd:''
				}
		    	vm.editErrorMsg = '';
				vm.$refs.editIpForm.resetFields();
		    	vm.editIpShow = false;
		    },
			//ip list 表格删除
			deleteIp(row,index){ 
				var vm = this;
				vm.$confirm('<%= rb.getString("QueRenShanChu")%>','<%= rb.getString("QueRen")%>').then(function(r){
                    if(r) {
                    	if(index >= 0) {
        					vm.ipTableData.splice(index,1);
        			    }
                    }
                }).catch(function(){});
                
				vm.editIpShow = false;
				vm.addIpShow = false;
				vm.showImportCard = false;
		    },
			
			//添加ip
			addIpBtn(){
		    	var vm = this,
					reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
					ipStart = vm.addIpForm.ipStart,
					ipEnd = vm.addIpForm.ipEnd,
					str = '';
				
				//endIp不是必填项
				if(ipStart == ''){					
					vm.errorMessage = '<%=rb.getString("IPDiZhiBuNengWeiKong")%>';
				}else{
					//如果只有start IP
					if(ipEnd == '' || ipEnd == null){
						if(reg.test(ipStart)){
							str = ipStart;
							if(vm.addIpForm.ipGroup.indexOf(str) == -1){
								vm.addIpForm.ipGroup.push(str);
								vm.addIpForm.ipStart = '';
								vm.errorMessage = '';
							}else{
								vm.errorMessage = '<%=rb.getString("IPYiCunZai")%>';
							}
						}else{
							vm.errorMessage = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}else{
						//如果star ip ，end ip都有
						if(reg.test(ipStart) && reg.test(ipEnd) && vm.compareIp(ipStart,ipEnd)){
							str = ipStart + "-" + ipEnd;
							if(vm.addIpForm.ipGroup.indexOf(str) == -1){
								vm.addIpForm.ipGroup.push(str);
								vm.addIpForm.ipStart = '';
								vm.addIpForm.ipEnd = '';
								vm.errorMessage = '';
							}else{
								vm.errorMessage = '<%=rb.getString("IPFanWeiYiCunZai")%>';
							}
						}else{
							vm.errorMessage = '<%=rb.getString("IPDiZhiFeiFa")%>';
						}
					}										
				} 
			},
			/**
			 * 新建ip弹窗中的单个删除ip
			 * @param item {string} 删除项
			*/
			removeIp(item){
				var vm = this, index = vm.addIpForm.ipGroup.indexOf(item);				
				if(index !== -1){
					vm.addIpForm.ipGroup.splice(index,1)
				}
				vm.errorMessage = '';
			},
			/**
			 * 比较ip大小
			 * @param ipStart {string} 初始ip
			 * @param ipEnd {string} 结束ip
			*/
			compareIp(ipStart,ipEnd){
				var temp1,
					temp2,
					bool = true;
				temp1 = ipStart.split(".");
				temp2 = ipEnd.split(".");
				for(var i=0;i<4;i++){
					if(parseInt(temp1[i])>parseInt(temp2[i])){
						bool = false;
					}
				}

				return bool;
			},
			
			//添加 ip
			saveAddIp(){
				var vm = this;
				if(vm.addIpForm.ipGroup.length == 0){
					vm.errorMessage = '<%=rb.getString("ZhiShaoTianJiaYiGe")%>';
				}else{
					var ipListData = vm.addIpForm.ipGroup.toString().split(",").join(";"),									
						newIpListData = ipListData.split(';'),
						getRowData ={};
					
					newIpListData.map((item,index) => {
						if(item.includes('-')){
							getRowData = {     						
								ipStart:item.split('-')[0],
	      						ipEnd:item.split('-')[1]	    			
	      		    		}
						}else{
							getRowData = {     						
								ipStart:item	    			
		      		    	}
						}
						vm.ipTableData.push(getRowData);
					}) 
					vm.oldNewIpTableData();
					vm.closeAddIp();
				}				
			},
			
			oldNewIpTableData(){
				var vm = this;
				for(var i = 0; i< vm.ipTableData.length; i++){
					for(var j = i+1; j < vm.ipTableData.length; j++ ){
						//两种情况，1：只有 start ip; 2 start ip, end ip 都有时
						if(vm.ipTableData[i].ipEnd){
							if(vm.ipTableData[i].ipStart == vm.ipTableData[j].ipStart && vm.ipTableData[i].ipEnd == vm.ipTableData[j].ipEnd){
								vm.ipTableData.splice(j,1);								
								j--;
							}
						}else{
							if(vm.ipTableData[i].ipStart == vm.ipTableData[j].ipStart){
								vm.ipTableData.splice(j,1);							
								j--;
							}
						}					
					}					
				}
				return vm.ipTableData;
			},
			
			//关闭添加IP弹窗
			closeAddIp(){
				var vm = this;
				vm.addIpForm = {
					ipStart:'',
					ipEnd:'',
					ipGroup:[]
				}
				vm.errorMessage = '';
				vm.$refs.addIpForm.resetFields();
				vm.addIpShow = false;			
			},
			
			//wifi 开关改变
			wlanEnableChange(val) {
				var vm = this,
				row = vm.tbData[0];
				
				if(row) {
					row.wifiEnable = val == '1'?'1':'0';
				}
			},	
			wifi5EnableChange(val) {
				var vm = this,
					row = vm.tbDataWifi5[0];
				
				if(row) {
					row.wifi5WifiEnable = val == '1'?'1':'0';
				}
			},
			
			// 关闭
			closePanel(){
				var vm = this;
				if(vm.operateType == 'information'){
					eventBus.$emit('hide-slide');
				}else{
					vm.cancel();
				}				
			}, 

            closeConfig() {
                eventBus.$emit('close-config');
            },
            //产品型号改变,需更新目标版本数据
            productChange(val) {
                var vm = this;
                vm.addOrEditForm.targetVersion  = '';
                vm.commonTargetVerList();
                vm.commonOriginalVerList();
            },
            
            //新建或修改 policy
            cpeAddOrUpdateSubmit() {
            	var vm = this, curRangeIp = '', curPciParams = '', curForm = vm.addOrEditForm, scanMode = vm.addOrEditForm.scanMode,
		            url = '${ctx}/plugandplay/policy/addAutoCPEPolicy.action',
		            params = {};
            	if(vm.curType == 'modify'){
            		params.policyId = vm.curPolicyId;
            	}else{
            		params.policyId = '';
            	}
		        params.policyName = encodeURIComponent(vm.addOrEditForm.policyName);
				params.productModule = vm.addOrEditForm.productModule.join(',');
				//params.pushType = vm.addOrEditForm.pushType;
				params.targetVersion = vm.addOrEditForm.targetVersion;
				//选中 Any 时，参数传 'allExecute'
				 //params.specifyVersionType = vm.addOrEditForm.specifyVersionType;
                if(vm.addOrEditForm.specifyVersionType == '1'){
                	params.specifyVersionType = 'allExecute';
                	vm.resultOriginalVersionList = [];
                }else{
                	params.specifyVersionType = '';
                }
                params.upgradeEnable = vm.addOrEditForm.upgradeEnable;
				params.configEnable = vm.addOrEditForm.configEnable;
		      
		        var flagEarfcn = false;	
				//earfcn, earfcn and pci, pci 参数配置开关打开时 校验
				if(vm.addOrEditForm.configEnable == true){
					if(vm.addOrEditForm.scanMode == 'freqpreferred'){
						if(vm.addOrEditForm.earfcnList.length == 0){
							flagEarfcn = true;
							vm.earfcnErrorMessage = '<%=rb.getString("PinDianWeiKong")%>';
						}else{
							flagEarfcn = false;
							vm.earfcnErrorMessage = '';						
						} 
					}else if(vm.addOrEditForm.scanMode == 'pcilock'){
						if(vm.addOrEditForm.earfcnPciList.length == 0){
							flagEarfcn = true;
							vm.earfcnPciErrorMessage = '<%=rb.getString("PinDianWeiKong")%> ';
						}else{
							flagEarfcn = false;
							vm.earfcnPciErrorMessage = '';						
						}
					}else if(vm.addOrEditForm.scanMode == 'pcionlylock'){
						if(vm.addOrEditForm.pciList.length == 0){
							flagEarfcn = true;
							vm.pciErrorMessage = '<%=rb.getString("PCIWeiKong")%>';
						}else{
							flagEarfcn = false;
							vm.pciErrorMessage = '';						
						}
					}
				}else{
					vm.earfcnErrorMessage = '';	
					vm.earfcnPciErrorMessage = '';		
					vm.pciErrorMessage = '';
				}
							
					//earfcn
					if(scanMode == 'freqpreferred') {
						curPciParams = curForm.earfcnList.map(function(item){
							return item;							
						}).join(';');						
					}
					// earfcn and pci
					if(scanMode == 'pcilock') {
						curPciParams = curForm.earfcnPciList.map(function(item){
							var curPci = item.startValue +','+ item.endValue;						
							return curPci;
						}).join(';');
					}
					//pci
					if(scanMode == 'pcionlylock') {
						curPciParams = curForm.pciList.map(function(item){
							return item;
						}).join(';');
					}
					vm.tbData.map(function(row){
						if(['1','2','3'].includes(row.id)) {
							var index = row.id,
								pre = 'wifi',
								enableKey = pre+index+'Enable',
								idKey = pre+index+'Ssid',
								encryKey = pre+index+'Encryption',
								phraKey = pre+index+'Passphrase';							
							curForm[enableKey] = row['wifiEnable'];
							curForm[idKey] = row['wifiSsid'];
							curForm[encryKey] = row['wifiEncryption'];
							curForm[phraKey] = row['wifiPassphrase'];
						}else {
							Object.assign(curForm,{
								wifiSsid: row['wifiSsid'],
								wifiEncryption: row['wifiEncryption'],
								wifiPassphrase: row['wifiPassphrase']
							});
						}
					});
					vm.tbDataWifi5.map(function(row){
						if(['1','2','3'].includes(row.id)) {
							var index = row.id,
								pre = 'wifi5Wifi',
								enableKey = pre+index+'Enable',
								idKey = pre+index+'Ssid',
								encryKey = pre+index+'Encryption',
								phraKey = pre+index+'Passphrase';
							curForm[enableKey] = row['wifi5WifiEnable'];
							curForm[idKey] = row['wifi5WifiSsid'];
							curForm[encryKey] = row['wifi5WifiEncryption'];
							curForm[phraKey] = row['wifi5WifiPassphrase'];
						}else {
							Object.assign(curForm,{
								wifi5WifiSsid: row['wifi5WifiSsid'],
								wifi5WifiEncryption: row['wifi5WifiEncryption'],
								wifi5WifiPassphrase: row['wifi5WifiPassphrase']
							});
						}
					});
					var paramConfig= {
						wifi:{
							wlanEnable : vm.addOrEditForm.wlanEnable,
							wifiChannel: vm.addOrEditForm.wifiChannel,
							wifiBandwidth : vm.addOrEditForm.wifiBandwidth,
							
							wifiSsid: vm.addOrEditForm.wifiSsid,
							wifiEncryption : vm.addOrEditForm.wifiEncryption,
							wifiPassphrase: vm.addOrEditForm.wifiPassphrase,
							
							wifi1Enable : vm.addOrEditForm.wifi1Enable,
							wifi1Ssid: vm.addOrEditForm.wifi1Ssid,
							wifi1Encryption: vm.addOrEditForm.wifi1Encryption,
							wifi1Passphrase : vm.addOrEditForm.wifi1Passphrase,
							
							wifi2Enable: vm.addOrEditForm.wifi2Enable,
							wifi2Ssid : vm.addOrEditForm.wifi2Ssid,
							wifi2Encryption: vm.addOrEditForm.wifi2Encryption,
							wifi2Passphrase: vm.addOrEditForm.wifi2Passphrase,
							
							wifi3Enable: vm.addOrEditForm.wifi3Enable,
							wifi3Ssid : vm.addOrEditForm.wifi3Ssid,
							wifi3Encryption: vm.addOrEditForm.wifi3Encryption,
							wifi3Passphrase: vm.addOrEditForm.wifi3Passphrase
						},
						wifi5:{
							wifi5WlanEnable : vm.addOrEditForm.wifi5WlanEnable,
							wifi5WifiMode: vm.addOrEditForm.wifi5WifiMode,
							wifi5WifiChannel : vm.addOrEditForm.wifi5WifiChannel,
							wifi5WifiBandwidth : vm.addOrEditForm.wifi5WifiBandwidth,
							wifi5WifiSupportChannel : vm.addOrEditForm.wifi5WifiSupportChannel,
							
							wifi5WifiSsid: vm.addOrEditForm.wifi5WifiSsid,
							wifi5WifiEncryption : vm.addOrEditForm.wifi5WifiEncryption,
							wifi5WifiPassphrase: vm.addOrEditForm.wifi5WifiPassphrase,
							
							wifi5Wifi1Enable : vm.addOrEditForm.wifi5Wifi1Enable,
							wifi5Wifi1Ssid: vm.addOrEditForm.wifi5Wifi1Ssid,
							wifi5Wifi1Encryption: vm.addOrEditForm.wifi5Wifi1Encryption,
							wifi5Wifi1Passphrase : vm.addOrEditForm.wifi5Wifi1Passphrase,
							
							wifi5Wifi2Enable: vm.addOrEditForm.wifi5Wifi2Enable,
							wifi5Wifi2Ssid : vm.addOrEditForm.wifi5Wifi2Ssid,
							wifi5Wifi2Encryption: vm.addOrEditForm.wifi5Wifi2Encryption,
							wifi5Wifi2Passphrase: vm.addOrEditForm.wifi5Wifi2Passphrase,
							
							wifi5Wifi3Enable: vm.addOrEditForm.wifi5Wifi3Enable,
							wifi5Wifi3Ssid : vm.addOrEditForm.wifi5Wifi3Ssid,
							wifi5Wifi3Encryption: vm.addOrEditForm.wifi5Wifi3Encryption,
							wifi5Wifi3Passphrase: vm.addOrEditForm.wifi5Wifi3Passphrase
						},
						dmz:{
							dmzEnable : vm.addOrEditForm.dmzEnable,
							dmzHostAddress: vm.addOrEditForm.dmzHostAddress
						},
						lan:{
							lanEnable : vm.addOrEditForm.lanEnable,
							lanDHCPEnable: vm.addOrEditForm.lanDHCPEnable,
							lanDNS1 : vm.addOrEditForm.lanDNS1,
							lanDNS2: vm.addOrEditForm.lanDNS2,
							lanOption: vm.addOrEditForm.lanOption
						},
						pciLock:{
							scanMode: vm.addOrEditForm.scanMode,
							//pci: vm.addOrEditForm.pci
							pci: curPciParams
						},
						wanAcc:{
							uiPassword : vm.addOrEditForm.uiPassword,
							httpsEnable: vm.addOrEditForm.httpsEnable,
							httpsWanEnable : vm.addOrEditForm.httpsWanEnable,
							wanAccEnable: vm.addOrEditForm.wanAccEnable,
							rangeIp: vm.addOrEditForm.rangeIp
							//rangeIp: curRangeIp
						},
						watchDog:{
							watchDogEnable : vm.addOrEditForm.watchDogEnable,
							watchDogPingIp: vm.addOrEditForm.watchDogPingIp,
							watchDogPingTimeout : vm.addOrEditForm.watchDogPingTimeout,
							watchDogPingCount: vm.addOrEditForm.watchDogPingCount,
							watchDogFailureReboot: vm.addOrEditForm.watchDogFailureReboot
						}
					}
				params.paramConfig = JSON.stringify(paramConfig);
				vm.$refs.addOrEditForm.validate(function(valid){
					if(valid && flagEarfcn == false){

						if(vm.resultOriginalVersionList.length > 0){
            				params.orginalVersion = vm.resultOriginalVersionList.map(function(item){ return item.orginalVersion;}).join(',');
            			}
						//if(isFormChanged(vm.$refs.addOrEditForm)) {
							axios.post(url, stringify(params)).then(function(res){
		                        var data = res.data;
		
		                        if(data['success'] == true) {
		                            vm.$message({
			    						message: '<%=rb.getString("ChengGong")%>',
			    						type:'success',
			    					});
                                    eventBus.$emit('reload-config-list');
		                            vm.closeConfig();
		                        }else {
		                            vm.$message.error(data['message']);
		                        }
		                    }).catch(function(){});
						//}						
		            }
		        });
		    },
		    
			
            cpeAddOrUpdateCancel(){
            	var vm = this;
				eventBus.$emit('cancel-plugPlaySlide')
            },
            closeConfig() {
                eventBus.$emit('close-config');
            },
            queryEnb(text) {
				this.enbQuery.searchText = text;
			},
			
        },
        mounted() {
        	var vm = this;
            eventBus.$off('save-plugPlayConfig').$on('save-plugPlayConfig', vm.cpeAddOrUpdateSubmit);
            eventBus.$off('handle-plugPlayCancel').$on('handle-plugPlayCancel',vm.cpeAddOrUpdateCancel);
            eventBus.$off('init-plugPlayConfig').$on('init-plugPlayConfig', vm.cpeInit);
            //this.init()
        }
    });
</script>

