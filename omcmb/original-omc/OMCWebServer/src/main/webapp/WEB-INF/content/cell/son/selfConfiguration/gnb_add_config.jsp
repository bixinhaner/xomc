<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#gnbAddOrEditConfigPage { background: #F6F7FB; position: relative; width: 100%; height: 100%; overflow: hidden; }
	#gnbAddOrEditConfigPage .gnbWarp { height: calc(100% - 2px);  width: 100%; }
	#gnbAddOrEditConfigPage .leftWarp { flex: 1; height: 100%; position: relative; overflow: hidden; flex-direction: column; border-radius: 10px; border: 1px solid #D5DCEC; background: #FFFFFF; }
	#gnbAddOrEditConfigPage .mainContent { width: 100%; overflow-y: scroll; height: 100%; position: absolute; top: 40px; z-index: 0; box-sizing: border-box; bottom: 70px; height: auto; }
	#gnbAddOrEditConfigPage .mainContent .el-form-item { margin-bottom: 22px; }
	#gnbAddOrEditConfigPage .mainContent .el-form-item__label { line-height: 28px; }
	#gnbAddOrEditConfigPage .titleBox {  width: 100%;  height: 38px;  line-height: 38px;  position: absolute;  top: 0;  left: 0;  z-index: 100; border-bottom: 1px solid #D5DCEC; background: #FFFFFF; }
	#gnbAddOrEditConfigPage .slideTopTitle { padding: 0 30px; }
	#gnbAddOrEditConfigPage .newIconBoxCls-bt { right: 15px; top: 8px; }
	#gnbAddOrEditConfigPage .newIconBoxCls-bt .el-icon-circle-close:before { content: '\e778'; font-size: 12px !important; }
	#gnbAddOrEditConfigPage .basicInfoBox { margin: 16px 40px 10px; }
	#gnbAddOrEditConfigPage .basicInfo { padding-left: 26px;  padding-top: 16px; }
    #gnbAddOrEditConfigPage .split-line { border: none; border-top: 1px solid #D5DCEC; margin: 10px 0 20px 0; }
    #gnbAddOrEditConfigPage .el-select-mini input { min-height: 26px; }
	#gnbAddOrEditConfigPage .commonDisplayBlock { display: block; }
	#gnbAddOrEditConfigPage .originalBox .el-checkbox { padding: 3px 6px 0 10px; }
	#gnbAddOrEditConfigPage .addVersionWarp { width: 400px; height: 254px; border-radius: 4px; text-align: center; border: 1px solid #D5DCEC;}
	#gnbAddOrEditConfigPage .addVersionBtn { font-size: 20px; padding: 100px 0 10px; }
	#gnbAddOrEditConfigPage .selectVersionBox .el-form-item__content { margin-left: 0px !important; }
	#gnbAddOrEditConfigPage .selectVersionBox .el-form-item__content .el-input { width: 300px; }
	#gnbAddOrEditConfigPage .selectMethodBox { display: flex; flex-direction: row; justify-content: space-between; width: 80%; }
	#gnbAddOrEditConfigPage .selectMethodBox .el-radio-button__inner { display: flex; border: 1px solid #D5DCEC; min-width: 240px; height: 50px; background: #F5F7FE; line-height: 50px; border-radius: 4px; padding: 0; font-size: unset; font-weight: normal;text-align: center;}
	#gnbAddOrEditConfigPage .selectMethodBox .el-radio-button__inner [class*=el-icon-]+span { margin-left: 0; }
	#gnbAddOrEditConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .methodTitle {color:var(--main-color);}
	#gnbAddOrEditConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .commonIconStyle,
	#gnbAddOrEditConfigPage .selectMethodBox .el-radio-button__inner:hover,#gnbAddOrEditConfigPage .selectMethodBox .el-radio-button__inner:hover .commonIconStyle,#gnbAddOrEditConfigPage .el-radio:hover, #gnbAddOrEditConfigPage .el-radio .el-radio__inner:hover { color:var(--main-color); background: rgba(var(--main-color-rgba1),0.1); border-color:var(--main-color) !important; }
	#gnbAddOrEditConfigPage .licenseBox .el-query{ right: 0; }
	.gnbIpsecConfigAddDialog .licenseImportBox, #gnbAddOrEditConfigPage .licenseImportBox { margin: 2px 10px; width: 26px; height: 26px; border: 1px solid #D7D7E6; text-align: center; border-radius: 8px; }
	.gnbIpsecConfigAddDialog .licenseImportBox i, #gnbAddOrEditConfigPage .licenseImportBox i { font-size: 12px; line-height: 26px; }
	#gnbAddOrEditConfigPage .footerBox { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
	#gnbAddOrEditConfigPage .closeIconBox .el-icon { font-size: 14px !important; color: #7a7992; margin: 12px 16px;}
	#gnbAddOrEditConfigPage .rightBox320 { flex: 0 320px; height: 100%;  margin-left: 10px; border-radius: 10px; overflow: hidden; border: 1px solid #E9EDF9;background: #FFFFFF;}
	#gnbAddOrEditConfigPage .rightBox { flex: 0 1 360px; height: 100%;  margin-left: 10px; border-radius: 10px; overflow: hidden; border: 1px solid #D5DCEC;background: #FFFFFF;}
	#gnbAddOrEditConfigPage .rightBox .commonRightTitleBox {height: 38px; line-height: 38px; display: flex; justify-content: space-between; margin: 0px 20px; border-bottom: 1px solid #D5DCEC; }
	#gnbAddOrEditConfigPage .rightBox .commonRightTitleBox .el-icon { margin: 12px 0; color:#7A7992; }
	.gnbImportConfigAddDialog .titleAndNoteBox .el-form-item__label, #gnbAddOrEditConfigPage .rightBox .titleAndNoteBox .el-form-item__label { display: flex; }
	#gnbAddOrEditConfigPage .commonTitle { height: 38px; line-height: 38px; }
	#gnbAddOrEditConfigPage .rightHeaderBox .addTitle { padding: 0; }
	#gnbAddOrEditConfigPage .rightTextBox { padding: 10px 0; border-bottom: 1px solid #D5DCEC; }
	#gnbAddOrEditConfigPage .commonBorderTop { border-top: 1px solid #D5DCEC; }
	#gnbAddOrEditConfigPage .originalVersionBox { padding: 16px 0 0; }
	#gnbAddOrEditConfigPage .originalVersionBox .el-input{ width: 310px }
	#gnbAddOrEditConfigPage .width280 .el-input{ width: 280px; }
	#gnbAddOrEditConfigPage .originalVersionBox .el-input__inner{ height: 30px; line-height: 30px; }
	#gnbAddOrEditConfigPage .originalVersionBox .el-input-group__append { padding: 0 8px; background-color: #FFFFFF; }
	#gnbAddOrEditConfigPage .originalVersionBox .el-input-group__append .el-icon { font-size: 14px; }
	#gnbAddOrEditConfigPage .originalVersionBox .el-input-group__append .el-icon:before { color: #7A7992; }
	#gnbAddOrEditConfigPage .originalVersionBox .el-table tr { display: none; }
	#gnbAddOrEditConfigPage .ipErrorTip { color: #FA5555; font-size: 12px; }
	#gnbAddOrEditConfigPage .versionResultBox { border-radius: 4px; background-color: #FFFFFF; margin-top: 5px; width: 318px; max-height: 122px; padding: 5px 0; overflow: auto; border: 1px solid #D5DCEC;}
	#gnbAddOrEditConfigPage .versionResultBox .el-form-item { margin-right: 0 !important;}
	#gnbAddOrEditConfigPage .form-suffix { position: relative; margin: 3px 0 0 20px; padding: 0; width:278px; border: none; background: #fff; }
	#gnbAddOrEditConfigPage .form-suffix:hover { background: #F4F9FF; border-radius: 100px; }
	#gnbAddOrEditConfigPage .form-suffix .deleteVersion { display: none; position: absolute; right: 0; top: 0px; color: var(--main-color); font-size: 14px;}
	#gnbAddOrEditConfigPage .form-suffix:hover .deleteVersion { display: inline-block !important; }
	#gnbAddOrEditConfigPage .form-suffix .text { padding-left: 10px; color: #666666; }
	#gnbAddOrEditConfigPage .selectedVersion .queryGroup .el-input__inner,
	#gnbAddOrEditConfigPage .curImportQuery .el-input__inner { width: 220px !important; }
	#gnbAddOrEditConfigPage .selectedVersion .el-query .advanceQuery { height: 28px; }
	#gnbAddOrEditConfigPage .selectedVersion .el-query .advanceQuery .el-input.el-input--small { width: 230px !important; }
	#gnbAddOrEditConfigPage .selectedVersion .el-query { right: 0px; }
	#gnbAddOrEditConfigPage .operBtn { width: 26px; height: 26px; border: 1px solid #D7D7E6; border-radius: 8px; margin-top: 1px; }
	#gnbAddOrEditConfigPage .operBtn .el-icon { line-height: 26px; }
	#gnbAddOrEditConfigPage .operBtn .el-icon:before, #gnbAddOrEditConfigPage .importIcon:before { color: #7A7992; }
	#gnbAddOrEditConfigPage .el-radio.is-bordered { max-width: 160px; height: 30px; padding: 7px 12px; }
	#gnbAddOrEditConfigPage .el-radio-group .el-radio__label { font-size: 12px; }
	#gnbAddOrEditConfigPage .enableCommon .el-switch { margin-top: 4px; }
	#gnbAddOrEditConfigPage .selectCommon .el-select .el-input { width: 150px; }
	#gnbAddOrEditConfigPage .inputCommon .el-input__inner { border-radius: 4px; }
    #gnbAddOrEditConfigPage .paramTwoBox { padding: 10px 20px; }
	#gnbAddOrEditConfigPage .infoSpecifiedDevice .el-collapse-item__arrow { position: absolute; left: 0; top: 0px; }
    #gnbAddOrEditConfigPage .infoSpecifiedDevice .el-collapse-item {  position: relative; }
    #gnbAddOrEditConfigPage .infoSpecifiedDevice .el-icon-arrow-right {  font-size: 16px; }
    #gnbAddOrEditConfigPage .infoSpecifiedDevice .el-icon-arrow-right:before { content: "\e639"; color: #BBB;  }
    #gnbAddOrEditConfigPage .infoSpecifiedDevice .is-active.el-icon-arrow-right:before { content: "\e638"; color: #BBB; }
    #gnbAddOrEditConfigPage .infoSpecifiedDevice .el-collapse-item__arrow.is-active { transform: rotate(0deg); }
    #gnbAddOrEditConfigPage .infoSpecifiedDevice .el-collapse { border-top: 1px solid #fff; border-bottom: 1px solid #fff; }
    #gnbAddOrEditConfigPage .infoSpecifiedDevice .el-collapse-item__wrap { border-bottom: 1px solid #fff; }
    #gnbAddOrEditConfigPage .infoSpecifiedDevice .btn-next .el-icon-arrow-right:before { content: "\e794"; }
	#gnbAddOrEditConfigPage .el-collapse-item__content { padding-bottom: 0; }
	#gnbAddOrEditConfigPage .reginDeployingBox .el-tabs--border-card { border: 1px solid #D5DCEC; box-shadow: none; -webkit-box-shadow: none; }
	#gnbAddOrEditConfigPage .reginDeployingBox .el-tabs .el-tabs__header, #gnbAddOrEditConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header { border-bottom: 1px solid #D5DCEC; }
	#gnbAddOrEditConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header { background-color: #FFFFFF; }
	#gnbAddOrEditConfigPage .reginDeployingBox .el-tabs__item { font-size: 14px; color: #7A7992; font-weight: normal;  }
	#gnbAddOrEditConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header .el-tabs__item.is-active { color: #7A7992; font-weight: bold;  border-right-color: #D5DCEC; border-left-color: #D5DCEC; }
	#gnbAddOrEditConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header .el-tabs__item:not(.is-disabled):hover { color: #4D84FF; }
    #gnbAddOrEditConfigPage .table-suffix { border: none; position: relative; }
	#gnbAddOrEditConfigPage .table-suffix .deleteVersion { display: none; position: absolute; right: 0; top: 2px; color : var(--main-color); font-size: 14px;}
	#gnbAddOrEditConfigPage .table-suffix:hover .deleteVersion { display: inline-block !important; }
	#gnbAddOrEditConfigPage .table-suffix .text { padding-left: 10px; color: #666666; }
	#gnbAddOrEditConfigPage .disabledClass { cursor: not-allowed !important; opacity: 0.4; }
	#gnbAddOrEditConfigPage .disabledClass:before {color: #c0c4cc;  cursor: not-allowed !important; }
	#gnbAddOrEditConfigPage .defaultClass { cursor: pointer; }
	#gnbAddOrEditConfigPage .paramPoolWarp .el-form-item { width: 49%; display: inline-block; flex-direction: row; }
	#gnbAddOrEditConfigPage .paramPoolWarp .el-form-item .el-input { width: 200px; }
	#gnbAddOrEditConfigPage .paramPoolWarpThree .el-form-item { width: 33%; min-width: 460px; display: inline-block; flex-direction: row; }
	#gnbAddOrEditConfigPage .basicLeftLabel { display: flex; }
	#gnbAddOrEditConfigPage .basicLeftLabel .el-form-item__label { width: 125px; }
	#gnbAddOrEditConfigPage .commonFormFotter { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
	#gnbAddOrEditConfigPage .el-form-item__error { padding-top: 0 !important; }
	#gnbAddOrEditConfigPage .el-input.is-disabled .el-input__inner { height: 26px !important; }
	#gnbAddOrEditConfigPage .contentHeight { height: calc(100% - 100px); overflow: scroll; }
	#gnbAddOrEditConfigPage .el-icon-menu-system:before { color: #7A7992; }
	#gnbAddOrEditConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .el-icon-menu-system:before,
	#gnbAddOrEditConfigPage .selectMethodBox .el-radio-button__inner:hover .el-icon-menu-system:before { color:var(--main-color); }
	#gnbAddOrEditConfigPage .is-error .el-input-group__append{ color:#FA5555; }
	#gnbAddOrEditConfigPage .commonIconStyle { margin: 10px 15px; width: 30px; height: 30px; color: #7A7992;background: rgba(122,121,146,0.1); border-radius: 100px; line-height: 30px !important; }
	#gnbAddOrEditConfigPage .commonContent { justify-content: space-between; }
	#gnbAddOrEditConfigPage .validateItem .el-input-group__append { border:none; background:none; padding: 0px 10px; }
	#gnbAddOrEditConfigPage .is-error .el-input-group__append { color:#FA5555; }
	#gnbAddOrEditConfigPage .validateItem .el-input__inner { width: 200px; }
	#gnbAddOrEditConfigPage .validateItem .el-input-group__append, #gnbAddOrEditConfigPage .validateRightItem .el-input-group__append { border:none; background:none; }
	#gnbAddOrEditConfigPage .validateItem .el-form-item__error, #gnbAddOrEditConfigPage .validateRightItem .el-form-item__error { display:none; }
	#gnbAddOrEditConfigPage .validateRightItem .el-input-group__append { border:none; background:none; padding: 0; }
	#gnbAddOrEditConfigPage .validateRightItem .el-input__inner { width:280px;}
	#gnbAddOrEditConfigPage .validateRightItem .el-input {display: block;} 
	.gnbIpsecConfigAddDialog .tieleTipCircle, #gnbAddOrEditConfigPage .tieleTipCircle { width: 8px; margin: 14px 10px 0 0; height: 8px; border-radius: 50%; background: rgba(0, 0, 0, 0.8); display: inline-block; }
	.gnbIpsecConfigAddDialog .contentTableTitle,#gnbAddOrEditConfigPage .contentTableTitle{display: flex;justify-content: space-between;width: 100%;}
	#gnbAddOrEditConfigPage .commonBorder { border: 1px solid #D5DCEC; border-radius: 10px; }
	.gnbImportConfigAddDialog .specialItemCls, #gnbAddOrEditConfigPage .specialItemCls{border: 1px solid #DFE2EE;position: relative;margin: 20px;padding: 10px 20px 0;border-radius: 8px;}
	.gnbImportConfigAddDialog .specialItemDelIcon, #gnbAddOrEditConfigPage .specialItemDelIcon{ height: 26px; width: 26px; display: flex; align-items: center; justify-content: center; border: 1px solid #DFE2EE; background-color: #FFFFFF; border-radius: 100px; position: absolute; right:-12px; top:-12px; z-index: 231 }
	.gnbIpsecConfigAddDialog .plmnLengthValidate .el-form-item__error {top: unset !important;padding-top: unset !important;}
	.gnbIpsecConfigAddDialog .gnbConfigAddMainBoxCls{height: 400px;width: 100%;position: relative;overflow-y: auto!important;overflow-x:hidden;}
	.gnbImportConfigAddDialog .el-icon-close {font-size: 16px !important;}
	.gnbIpsecConfigAddDialog .closeAddVlanCls .el-icon-close{font-size: unset;position: unset;top: unset;right: unset;}
	.gnbIpsecConfigAddDialog .inputAndSelect{ position: relative; }
	.gnbIpsecConfigAddDialog .inputAndSelect .el-select>.el-input{ width: 55px!important; }
	.gnbIpsecConfigAddDialog  .el-form-item { margin-bottom: 20px; }
	.gnbIpsecConfigAddDialog .inputAndSelect .el-select>.el-input .el-input__inner { width: 55px!important; }
	.gnbIpsecConfigAddDialog .inputAndSelect .el-input .el-input__inner { width: 145px !important; }
	.gnbIpsecConfigAddDialog .inputAndSelect .el-input-group__append { padding-left: 65px!important; }
	.gnbIpsecConfigAddDialog .el-dialog__header .el-icon:before { font-size: 16px; }
	.gnbIpsecConfigAddDialog .el-form { display: flex; flex-wrap: wrap; justify-content: space-between; width: 100%; position: relative; }
	.gnbIpsecConfigAddDialog .el-form-item { display: inline-block; width: 48%; }
	.gnbIpsecConfigAddDialog .el-form-item .el-form-item__label { font-size: 12px; }
	.gnbIpsecConfigAddDialog .el-form-item__error { padding-top: 0px; top:30px!important; }
	.gnbIpsecConfigAddDialog .selectErrcCls .el-form-item__error { position: absolute; left: 210px; top:5px !important; }
	.gnbIpsecConfigAddDialog  .validate-item .el-input__inner { width:200px; }
	.gnbIpsecConfigAddDialog .validate-item .el-input-group__append { border:none; background:none; padding: 0px 10px; }
	.gnbIpsecConfigAddDialog .validate-item .el-form-item__error { display:none; }
	.gnbIpsecConfigAddDialog .is-error .el-input-group__append { color:#FA5555; }
	.gnbIpsecConfigAddDialog .el-collapse-item__header { border-bottom:1px solid #fff; }
	.gnbIpsecConfigAddDialog .el-collapse-item__arrow { position:absolute; left:20px; top:0px; }
	.gnbIpsecConfigAddDialog .el-collapse-item { position:relative; border-bottom:1px solid #E9E9E9; }
	.gnbIpsecConfigAddDialog .el-collapse-item__content { margin: 0px 40px; padding-bottom: unset; }
	.gnbIpsecConfigAddDialog .el-collapse-item__header .el-icon-arrow-right { font-size:16px; }
	.gnbIpsecConfigAddDialog .el-collapse-item__header .el-icon-arrow-right:before { content:"\e639"; color:#BBB; }
	.gnbIpsecConfigAddDialog .el-collapse-item__header .is-active.el-icon-arrow-right:before { content:"\e638"; color:#BBB; }
	.gnbIpsecConfigAddDialog .el-collapse-item__arrow.is-active { transform:rotate(0deg); }
	.gnbIpsecConfigAddDialog .el-collapse { border-top:1px solid #fff; border-bottom:1px solid #fff; }
	.gnbIpsecConfigAddDialog .el-collapse { width: 100%; }
	.gnbIpsecConfigAddDialog .el-collapse-item__wrap{ border-bottom:1px solid #fff; padding-left: 0px; }
	.gnbIpsecConfigAddDialog .el-collapse-item__header{ max-width:800px; }
	.gnbIpsecConfigAddDialog .PlmnListItemCls{ border: 1px solid #DFE2EE; position: relative; margin-bottom: 20px; padding: 20px; }
	.gnbIpsecConfigAddDialog .plmnListDelIcon{ height: 26px; width: 26px; display: flex; align-items: center; justify-content: center; border: 1px solid #DFE2EE; background-color: #FFFFFF; border-radius: 100px; position: absolute; right:-12px; top:-12px; z-index: 231 }
	.gnbIpsecConfigAddDialog .sliceListBoxCls .el-collapse-item__content{ margin: 0px!important; }
	.gnbIpsecConfigAddDialog .sliceListBoxCls .el-collapse-item{ border-bottom:none!important; }
	.gnbImportConfigAddDialog .commonParamImportItem{ width: 33%; min-width: 280px; }
	.gnbIpsecConfigAddDialog .commonParamImportItem { width:40%; min-width:400px; }
	.gnbIpsecConfigAddDialog .sliceListBoxCls .plmnItemSliceListCls{ padding: 20px; width: 95%; border: 1px solid #DFE2EE; background-color: #FAFAFD; border-radius: 4px; position: relative; margin-bottom: 20px; }
	.gnbImportConfigAddDialog .allowMoreInputAddBtnCls,
	#gnbAddOrEditConfigPage .allowMoreInputAddBtnCls{ height: 26px; width: 56px; display: flex; align-items: center; justify-content: center; color:var(--main-color); border: 1px solid var(--main-color); background:rgba(var(--main-color-rgba1),0.1); border-radius: 4px; box-sizing: border-box; margin-left: 10px; cursor: pointer; }
	#gnbAddOrEditConfigPage .allowMoreInputAddBtnCls .el-icon::before{ font-size: 16px; color:var(--main-color); }
	#gnbAddOrEditConfigPage .allowMoreInputAddBtnCls span:nth-child(2){ margin: 0px 3px; }
	.gnbIpsecConfigAddDialog .commonParamBtn, #gnbAddOrEditConfigPage .commonParamBtn { border: 1px solid rgb(213, 220, 236); padding: 0 20px; border-radius: 4px; height: 22px; cursor: pointer; line-height: 22px; }
	#gnbAddOrEditConfigPage .commonText { display: flex; padding: 3px 0; justify-content: space-between;}
	#gnbAddOrEditConfigPage .marginRight5 { margin-right: 5px; }
	#gnbAddOrEditConfigPage .commonText1 { display: flex; justify-content: space-between; padding: 0 10px; }
	#gnbAddOrEditConfigPage .commonTextPlmn { display: inline-block; justify-content: space-between; padding: 0 10px; }
	#gnbAddOrEditConfigPage .tipLine { border-top: 1px solid #D5DCEC; width: calc(100% - 600px); line-height: 24px; height: 24px; display: none; margin-top: 10px; margin: 12px 10px; }
	#gnbAddOrEditConfigPage .el-table__expand-icon .el-icon-arrow-right::before{ content: "\e794" !important; }
	.container .slide-position-top .el-icon-close { font-size: 14px !important; }
	#gnbAddOrEditConfigPage .el-tabs--top .el-tabs__item.is-top:last-child { border-left: 1px solid #E9EDF9; }
</style>
<div id="gnbAddOrEditConfigPage">
   <div class='commonFlex gnbWarp'>
   		<div class='leftWarp'>
   			<div class='titleBox'>
   				<span class='slideTopTitle commonText14'>{{gnbPolicyTitle}}</span>
   				<div class="newIconBoxCls-bt" @click="gnbAddOrUpdateCancel">
					<span class="el-icon el-icon-circle-close"></span>
				</div>
   			</div>
   			<div class='mainContent'>
   				<el-form ref="gnbAddOrEditForm" :model="gnbAddOrEditForm" :rules="gnbAddOrEditRule" label-position="top" label-width="125">
                    <div class="basicInfoBox">
			        	<div class="group-title not-extend">
							<span class="title-icon"></span>
							<span class="title-text"><%=rb.getString("JiBenXinXi") %></span>
						</div>
						<div class='basicInfo'>
							<el-form-item label='<%=rb.getString("SheZhiKaiGuan") %>' prop='policySwitch' class='enableCommon basicLeftLabel'>
								<el-switch v-model="gnbAddOrEditForm.policySwitch" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="gnbIsReadOnly"></el-switch>
							</el-form-item>
							<el-form-item label='<%=rb.getString("CeLueMingChen") %>' prop="policyName" class='inputCommon basicLeftLabel'>
				                <el-input v-model="gnbAddOrEditForm.policyName" :disabled="gnbIsReadOnly" style="width: 300px;"></el-input>
				            </el-form-item>
				            <el-form-item label='<%=rb.getString("ChanPinLeiXingBiaoZhi") %>' prop="productType" placeholder='<%=rb.getString("QingXuanZe") %>' class='selectCommon basicLeftLabel'>
				                <el-select v-model="gnbAddOrEditForm.productType" :disabled="gnbIsReadOnly || gnbOperType=='modify'" @change="gnbProductTypeChange">
				                    <el-option v-for="item in gnbProductList" :label="item.name" :value="item.value"></el-option>
				                </el-select>
				            </el-form-item>
				            <el-form-item label='<%=rb.getString("ZhiXingFangShi") %>' prop="executeType" style='margin-bottom: 30px;' class='basicLeftLabel'>
				                <el-radio-group v-model="gnbAddOrEditForm.executeType" :disabled="gnbIsReadOnly">
				                    <el-radio label="0" border><%=rb.getString("eNBZiDongZhiXing") %></el-radio>
				                    <el-radio label="1" border style='margin-left: 20px;'><%=rb.getString("eNBShouDongZhiXing") %></el-radio>
				                </el-radio-group>
				            </el-form-item>
						</div>
					</div>
					<hr class="split-line"><!-- Software Upgrade,license model-->
					<div class='basicInfoBox'>
			        	<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 16px;'><%=rb.getString("KeYiXuanZeYiXiaMoKuaiJinXingPeiZhi") %></span>
			        	<div style="padding-bottom: 6px;">
			        		<el-radio-group v-model='gnbFunctionModulesSelect' class='selectMethodBox' @change='gnbSelectMethodChange'>
								<el-radio-button label="0" class='moduleConfigBox'>
									<i class='el-icon el-icon-status-upgrading commonIconStyle'></i><span class='commonTextNormal14  methodTitle'><%=rb.getString("RuanJianShengJi") %></span>
								</el-radio-button>
								<el-radio-button label="1" class='moduleConfigBox'>
									<i class='el-icon el-icon-operation-edit commonIconStyle'></i><span class='commonTextNormal14  methodTitle'>License</span>
								</el-radio-button>
								<el-radio-button label="2" class='moduleConfigBox'>
									<i class='el-icon el-icon-menu-system commonIconStyle'></i><span class='commonTextNormal14  methodTitle'><%=rb.getString("ENBCanShuZiPeiZhi") %></span>
			        			</el-radio-button>
							</el-radio-group>
			        	</div>
			        </div><!-- Software Upgrade -->
			        <div v-show="gnbFunctionModulesSelect == '0'" class="basicInfoBox commonColor" style='margin-top: 20px;'>
			            <div class="group-title not-extend">
							<span class="title-icon"></span><span class="title-text"><%=rb.getString("RuanJianShengJi") %></span>
							<el-switch v-model="gnbAddOrEditForm.upgradeEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="gnbIsReadOnly"></el-switch>
						</div>
						<div style='padding-left: 26px;'>
				            <span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 20px; margin-top: 14px;'><%=rb.getString("NinKeYiShouDongHuoLieBiaoXuanZeChuShiBanBenHao") %></span>
				            <div class='commonFlex originalBox'>
				            	<span class='commonSize14'><%=rb.getString("ChuShiBanBen") %></span>
				                <el-checkbox v-model="specifyVersionType" true-label="1" false-label="0" :disabled="gnbIsReadOnly"></el-checkbox>
				            	<span class='commonSize12ExportText' style='margin-top: 1px;'><%=rb.getString("SuoYouAny") %></span>
				            </div>
							<div class='commonFlex' style='padding: 10px 0 4px;'>
								<div class='addVersionWarp' v-show='gnbAddVersionBtnShow'>
									<div v-if="specifyVersionType == '1' || gnbIsReadOnly == true"><i class='el-icon el-icon-plus addVersionBtn commonDisplayBlock disabledClass'></i></div>
									<div v-else><i @click='gnbAddVersionClick' class='el-icon el-icon-plus addVersionBtn commonDisplayBlock defaultClass'></i></div>
									<span class='commonTitle12'><%=rb.getString("NinKeYiTianJiaYuanShiBanBen") %></span>
								</div>
								<div class='addVersionWarp' v-show='gnbResultOriginalVersionShow'>
									<el-ctable ref="resultOriginalVersionTable" row-key="originalVersion" class='commonBorderRadius' style='margin-bottom: 10px;'
										:data="gnbResultOriginalVersionList.filter(item=>{
											return item.originalVersion.indexOf(gnbSearchValue) > -1
										})" :pagination="false" :rownumber="false">
										<div slot="toolbar">
											<div class='commonFlex commonToolBarBox commonContent selectedVersion'>
												<div class='queryGroup' style='margin-right: 6px;'>
													<el-input v-model="gnbSearchValue" class='pairgrid-query' placeholder='<%=rb.getString("ChuShiBanBen")%>' style='width: 220px;'></el-input>
													<i class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
												</div>
												<div v-show="gnbOperType != 'view' && specifyVersionType == '0'" class='operBtn' @click='gnbClearVersionBtnClick' style='margin-right: 10px;'><i class='el-icon el-icon-operation-clear commonTextNormal12'></i></div>
												<div v-show="gnbOperType != 'view' && specifyVersionType == '0'" class='operBtn' @click='gnbAddVersionClick' style='margin-right: 10px;'><i class='el-icon el-icon-plus commonTextNormal12'></i></div>
											</div>
										</div>
 										<el-table-column label='<%=rb.getString("ChuShiBanBen") %>' prop="originalVersion" show-overflow-tooltip="true">
											<template slot-scope="scope">
												<div class='table-suffix'>
						                        	<span class="text">{{scope.row.originalVersion}} </span>
						                        	<span v-show="gnbOperType != 'view' && specifyVersionType=='0'"><i class='el-icon el-icon-circle-close ipTextWarp deleteVersion' @click='gnbDeleteVersionItem(scope.row)'></i></span>
						                        </div>
											</template>
										</el-table-column>
									</el-ctable>
								</div>
								<div style='padding: 60px 60px 0;'>
									<el-form-item label='<%=rb.getString("MuBiaoBanBen") %>' label-position="top" prop="targetVersion" class="selectVersionBox" style=" margin-bottom: 16px;">
					                    <el-select v-model="gnbAddOrEditForm.targetVersion" :disabled="gnbIsReadOnly" placeholder='<%=rb.getString("QingXuanZe") %>'>
					                       <el-option v-for="item in gnbTarVersionList" :key="item.text" :label="item.text" :value="item.text"></el-option>
					                    </el-select>
					                </el-form-item>
					                <el-checkbox v-model="gnbAddOrEditForm.preserveSetting" true-label="1" false-label="0" :disabled="gnbIsReadOnly"></el-checkbox>
					                <span class='commonSize14' style='margin-left: 6px;'><%=rb.getString("eNBBaoLiuPeiZhi") %></span>
								</div>
							</div>
							<el-form-item label-width='0' prop="originalVersion"><el-input v-model="gnbAddOrEditForm.originalVersion" v-show=false></el-input></el-form-item>
						</div>
			        </div><!-- License  -->
			        <div v-show="gnbFunctionModulesSelect == '1'" class="basicInfoBox commonColor" style='margin-top: 20px;'>
			            <div class="group-title not-extend">
							<span class="title-icon"></span><span class="title-text">License</span>
							<el-switch v-model="gnbAddOrEditForm.licenseEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="gnbIsReadOnly"></el-switch>
						</div>
			            <div style='height: 334px; padding-left: 26px;'>
							<el-ctable :height="height" ref="licenseTable" id='licenseTable' :url="gnbLicenseUrl" :time="6" :row-key="'serial_number'" class='commonBorderRadius commonBorder' style='margin-top: 10px; margin-bottom: 10px;' :pagination="true" :query-params="gnbLicenseQuery" row-key="serial_number">
								<div slot="toolbar">
									<div class='commonFlex commonToolBarBox commonContent licenseBox'>
										<span class='commonTitle12' style='padding: 6px 20px;'><%=rb.getString("GNBDaoRuLicenseFile")%></span>
										<div class='commonFlex'>
											<el-query type="normal" @query="queryEnbLicense" placeholder='<%=rb.getString("XiaoZhanBianMa")%>' style='margin-right: 10px;'></el-query>
											<div v-show='gnbOperType != "view"' class='licenseImportBox' @click='gnbImportLicenseClick'><i class='el-icon el-icon-operation-import importIcon'></i></div>
										</div>
									</div>
								</div>
								<el-table-column width="40" v-if='gnbOperType != "view"'>
									<template slot-scope="scope">
										<div><i @click="gnbDeleteLicenseClick(scope.row, event)" class="el-icon el-icon-operation-delete commonIcon"></i></div>
									</template>
								</el-table-column>
								<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
								<el-table-column label='<%=rb.getString("LicenseWenJian")%>' prop="file_name"></el-table-column>
								<el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' prop="upload_time"></el-table-column>
								<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="execute_status" :formatter="executeStatus"></el-table-column>
							</el-ctable>
			            </div>
			        </div><!-- Parametes Configuration -->
                 	<div v-show="gnbFunctionModulesSelect == '2'" class="basicInfoBox commonColor" style='margin-top: 20px;'>
			            <div class="group-title not-extend">
							<span class="title-icon"></span><span class="title-text"><%=rb.getString("ENBCanShuZiPeiZhi") %></span>
							<el-switch v-model="gnbAddOrEditForm.selfConfigEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="gnbIsReadOnly"></el-switch>
						</div>
			            <div style='padding-left: 26px; padding-top: 16px; '>
                            <span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 20px; margin-top: -6px;'><%=rb.getString("GNBCanShuPeiZhiTiShi") %></span>
                            <div class='reginDeployingBox infoSpecifiedDevice' style=' width: 100%; padding-bottom: 100px;'>
                                <el-tabs v-model="gnbParameterConfigActive" @tab-click="gnbClickTab" type='border-card' style='height: auto' >
                                    <el-tab-pane label='<%=rb.getString("CanShuShuJuChi") %>' name="gnbParamsConfig" style="position:relative">
                                        <div class='paramTwoBox'>
											<el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='paramConfigEnable' label-width='80px' class="enableCommon basicLeftLabel" style='margin-bottom: 0;'>
												<el-switch v-model="gnbAddOrEditForm.paramConfigEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="gnbIsReadOnly"></el-switch>
											</el-form-item>
											<el-collapse v-model="gnbActiveName">
												<el-collapse-item name="gnb">
													<template slot='title'><p style="display:inline-block;margin-left: 26px;"> <span class='commonText14'>GNB</span> </p></template>
													<div style='margin-left: 26px; margin-right: 0px;'>
														<div style='width: 100%;' class='paramPoolWarp'>
															<el-form-item label='<%=rb.getString("GNBMingCheng")%>' prop="gnbName" class='inputCommon validateItem'>
																<el-input v-model.trim="gnbAddOrEditForm.gnbName" maxlength="150" :disabled="gnbIsReadOnly">
																	<template slot="append"><%=rb.getString("GNBChangDuTiShi")%>1~150 Digit</template>
																</el-input>
															</el-form-item>
															<el-form-item label='<%=rb.getString("GNBBiaoShiChangDu")%>' prop="gnbLength" class='inputCommon validateItem'>
																<el-input v-model.trim="gnbAddOrEditForm.gnbLength" :disabled="gnbIsReadOnly">
																	<template slot="append"><%=rb.getString("FanWei")%>：22~32,<%=rb.getString("ZhengXing")%></template>
																</el-input>
															</el-form-item>
															<el-form-item label='<%=rb.getString("GNBBiaoShi")%>' prop="gnbId" class='inputCommon validateItem' style="width: 80%">
																<el-input v-model.trim="gnbAddOrEditForm.gnbId" :disabled="gnbIsReadOnly">
																	<template slot="append"><%=rb.getString("FanWei")%>：0~4294967295,<%=rb.getString("ZhengXing")%></template>
																</el-input>
																<span class='commonNotes12' style="display: block"><i class="el-icon el-icon-circle-info"></i><%=rb.getString("GNBCellZiDuanTiShi")%></span>
															</el-form-item>
														</div>
													</div>
												</el-collapse-item>
												<el-collapse-item name="cell"><!--CELL-->
													<template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonText14'>CELL</span></p></template>
													<div style='margin-left: 26px; margin-right: 0px;'>
														<div style='width: 100%;' class='paramPoolWarp'>
															<el-form-item label="PCI" prop="pci" class='inputCommon validateItem' style="width: 80%">
																<el-input v-model.trim="gnbAddOrEditForm.pci" :disabled="gnbIsReadOnly">
																	<template slot="append"><%=rb.getString("FanWei")%>：0~1007,<%=rb.getString("ZhengXing")%></template>
																</el-input>
																<span class='commonNotes12' style="display: block"><i class="el-icon el-icon-circle-info"></i><%=rb.getString("GNBCELLPCIZiDuanTiShi")%></span>
															</el-form-item>
														</div>
														<div>
															<div class="commonFlex" style="height: 24px; line-height: 24px;">
																<div @click="cellMoreItemShow = !cellMoreItemShow" style="cursor:pointer;">
																	<span class="commonText14"><%=rb.getString("GNBGengDuo")%></span>
																	<span class="el-icon el-icon-circle-down" :class="cellMoreItemShow ? 'el-icon-circle-up' : 'el-icon-circle-down'" style="padding: 0 10px;"></span>
																</div>
																<span v-show="gnbOperType != 'view'" @click="clearParamsValue('form')" class='commonParamBtn'><%=rb.getString("QingKong")%></span>
																<span v-show="gnbOperType != 'view'" @click="defaultParamsValue('form')" class='commonParamBtn' style='margin: 0 10px;'><%=rb.getString("APNMoRen")%></span>
																<span class="commonNotes12"><%=rb.getString("GNBZiDuanShuRuTiShi")%></span>
																<span class='tipLine'></span>
															</div>														
															<div style='width: 100%; padding-top: 10px;' class='paramPoolWarpThree' v-show="cellMoreItemShow">
																<el-form-item label='<%=rb.getString("GNBPinDaiZhiShi")%>' prop="freqBandIndicator" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.freqBandIndicator" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：1~1024,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("XiaXingNRARFCN")%>' prop="nrarfcndl" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.nrarfcndl" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("GNBXiaXingDaiKuan")%>' prop="dlbandwidth" class='inputCommon validateItem'>
																	<el-select v-model="gnbAddOrEditForm.dlbandwidth" :disabled="gnbIsReadOnly">
																		<el-option label="5MHz" value="5"></el-option>
																		<el-option label="10MHz" value="10"></el-option>
																		<el-option label="15MHz" value="15"></el-option>
																		<el-option label="20MHz" value="20"></el-option>
																		<el-option label="25MHz" value="25"></el-option>
																		<el-option label="30MHz" value="30"></el-option>
																		<el-option label="40MHz" value="40"></el-option>
																		<el-option label="50MHz" value="50"></el-option>
																		<el-option label="60MHz" value="60"></el-option>
																		<el-option label="80MHz" value="80"></el-option>
																		<el-option label="90MHz" value="90"></el-option>
																		<el-option label="100MHz" value="100"></el-option>
																		<el-option label="200MHz" value="200"></el-option>
																		<el-option label="400MHz" value="400"></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label='<%=rb.getString("SSBPinDianHao")%>' prop="ssbFrequency" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.ssbFrequency" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("ShangXingNRARFCN")%>' prop="nrarfcnul" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm .nrarfcnul" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("GNBShangXingDaiKuan")%>' prop="ulbandwidth" class='inputCommon validateItem'>
																	<el-select v-model="gnbAddOrEditForm.ulbandwidth" :disabled="gnbIsReadOnly">
																		<el-option label="5MHz" value="5"></el-option>
																		<el-option label="10MHz" value="10"></el-option>
																		<el-option label="15MHz" value="15"></el-option>
																		<el-option label="20MHz" value="20"></el-option>
																		<el-option label="25MHz" value="25"></el-option>
																		<el-option label="30MHz" value="30"></el-option>
																		<el-option label="40MHz" value="40"></el-option>
																		<el-option label="50MHz" value="50"></el-option>
																		<el-option label="60MHz" value="60"></el-option>
																		<el-option label="80MHz" value="80"></el-option>
																		<el-option label="90MHz" value="90"></el-option>
																		<el-option label="100MHz" value="100"></el-option>
																		<el-option label="200MHz" value="200"></el-option>
																		<el-option label="400MHz" value="400"></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label='<%=rb.getString("JiZhanZhiShi")%>' prop="duplex_mode" class='inputCommon validateItem'>
																	<el-select v-model='gnbAddOrEditForm.duplex_mode' :disabled="gnbIsReadOnly">
																		<el-option label='TDD' value='TDDMode'></el-option>
																		<el-option label="FDD" value='FDDMode'></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label='<%=rb.getString("GNBXiaXingZiZaiBoJianGe")%>' prop="dlSubcarrierSpacing" class='inputCommon validateItem'>
																	<el-select v-model='gnbAddOrEditForm.dlSubcarrierSpacing' :disabled="gnbIsReadOnly">
																		<el-option label="30kHz" value='1' v-if="gnbAddOrEditForm.duplex_mode == 'TDDMode'"></el-option>
																		<el-option label="15kHz" value='0' v-else></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label='<%=rb.getString("GNBShangXingXingZiZaiBoJianGe")%>' prop="ulSubcarrierSpacing" class='inputCommon validateItem'>
																	<el-select v-model='gnbAddOrEditForm.ulSubcarrierSpacing' :disabled="gnbIsReadOnly">
																		<el-option label="30kHz" value='1' v-if="gnbAddOrEditForm.duplex_mode == 'TDDMode'"></el-option>
																		<el-option label="15kHz" value='0' v-else></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label='<%=rb.getString("GNBFaSheTongDaoShu")%>' prop="ulAntNum" class='inputCommon validateItem'>
																	<el-select v-model='gnbAddOrEditForm.ulAntNum' :disabled="gnbIsReadOnly">
																		<el-option label="1" value='1'></el-option>
																		<el-option label='2' value='2'></el-option>
																		<el-option label="4" value='4'></el-option>																	
																	</el-select>
																</el-form-item>
																<el-form-item label='<%=rb.getString("GNBJieShouTongDaoShu")%>' prop="dlAntNum" class='inputCommon validateItem'>
																	<el-select v-model='gnbAddOrEditForm.dlAntNum' :disabled="gnbIsReadOnly">
																		<el-option label="1" value='1'></el-option>
																		<el-option label='2' value='2'></el-option>
																		<el-option label="4" value='4'></el-option>																		
																	</el-select>
																</el-form-item>
																<el-form-item label='<%=rb.getString("MoShi1ChunShuZhouQi")%>' prop="dlulTransmissionPeriodicity1" class='inputCommon validateItem'>
																	<el-select v-model='gnbAddOrEditForm.dlulTransmissionPeriodicity1' :disabled="gnbIsReadOnly">
																		<el-option label="0" value='0'></el-option>
																		<el-option label="1" value='1'></el-option>
																		<el-option label='2' value='2'></el-option>
																		<el-option label="3" value='3'></el-option>
																		<el-option label="4" value='4'></el-option>
																		<el-option label='5' value='5'></el-option>
																		<el-option label="6" value='6'></el-option>
																		<el-option label="7" value='7'></el-option>
																		<el-option label="8" value='8'></el-option>
																		<el-option label="9" value='9'></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label='<%=rb.getString("MoShi1LianXuXiaXingXiShu")%>' prop="nrofDownlinkSlots1" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.nrofDownlinkSlots1" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("MoShi1LianXuShangXingXiShu")%>' prop="nrofUplinkSlots1" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.nrofUplinkSlots1" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("MoShi1LianXuXiaXingFuHaoShu")%>' prop="nrofDownlinkSymbols1" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.nrofDownlinkSymbols1" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("MoShi1LianXuShangXingFuHaoShu")%>' prop="nrofUplinkSymbols1" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.nrofUplinkSymbols1" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("MoShi2ChunShuZhouQi")%>' prop="dlulTransmissionPeriodicity2" class='inputCommon validateItem'>
																	<el-select v-model='gnbAddOrEditForm.dlulTransmissionPeriodicity2' :disabled="gnbIsReadOnly">
																		<el-option label='<%=rb.getString("Guan")%>' value='4095'></el-option>
																		<el-option label="0" value='0'></el-option>
																		<el-option label="1" value='1'></el-option>
																		<el-option label='2' value='2'></el-option>
																		<el-option label="3" value='3'></el-option>
																		<el-option label="4" value='4'></el-option>
																		<el-option label='5' value='5'></el-option>
																		<el-option label="6" value='6'></el-option>
																		<el-option label="7" value='7'></el-option>
																		<el-option label="8" value='8'></el-option>
																		<el-option label="9" value='9'></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item v-if="gnbAddOrEditForm.dlulTransmissionPeriodicity2 != '4095'" label='<%=rb.getString("MoShi2LianXuXiaXingXiShu")%>' prop="nrofDownlinkSlots2" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.nrofDownlinkSlots2" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item v-if="gnbAddOrEditForm.dlulTransmissionPeriodicity2 != '4095'"  label='<%=rb.getString("MoShi2LianXuShangXingXiShu")%>' prop="nrofUplinkSlots2" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.nrofUplinkSlots2" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item v-if="gnbAddOrEditForm.dlulTransmissionPeriodicity2 != '4095'"  label='<%=rb.getString("MoShi2LianXuXiaXingFuHaoShu")%>' prop="nrofDownlinkSymbols2" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.nrofDownlinkSymbols2" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item v-if="gnbAddOrEditForm.dlulTransmissionPeriodicity2 != '4095'" label='<%=rb.getString("MoShi2LianXuShangXingFuHaoShu")%>' prop="nrofUplinkSymbols2" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.nrofUplinkSymbols2" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>														
																<el-form-item label='<%=rb.getString("GNBGenXuLieSuoYin")%>' prop="prachRootSequenceIndex" class='inputCommon validateItem'>
																	<el-select v-model='gnbAddOrEditForm.prachRootSequenceIndex' :disabled="gnbIsReadOnly">
																		<el-option label="0" value='0'></el-option>
																		<el-option label="1" value='1'></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label='<%=rb.getString("GNBGenXuLieZhi")%>' prop="prachRootSequenceValue" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.prachRootSequenceValue" :disabled="gnbIsReadOnly">
																		<template slot="append" v-if='gnbAddOrEditForm.prachRootSequenceIndex =="0"'><%=rb.getString("FanWei")%>：0~837,<%=rb.getString("ZhengXing")%></template>
																		<template slot="append" v-if='gnbAddOrEditForm.prachRootSequenceIndex =="1"'><%=rb.getString("FanWei")%>：0~137,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
																<span class='commonNotes12' style="display: block"><i class="el-icon el-icon-circle-info"></i><%=rb.getString("GNBZiDuanShuRuTiShi2")%></span>
															</div>
														</div>
													</div>
												</el-collapse-item>
												<el-collapse-item name="plmn"><!--PLMN-->
													<template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonText14'>PLMN</span></p></template>
													<div style='margin-left: 26px; margin-right: 0px;'>
														<div style='width: 100%;' class='paramPoolWarp'>
															<div class="contentTableTitle" style="padding-bottom: 5px;">
																<div>
																	<span class="commonSize14"><%=rb.getString("PLMNBiaoShiXinXiSheZhi")%></span>
																	<span class="commonNotes12" style="margin-left: 5px;"><%=rb.getString("BuChaoGuo6Ge")%></span>
																</div><!--row,operType,模块, 父级数据(plmn nr)-->
																<div class='licenseImportBox' style="margin: 0" v-show='gnbAddOrEditForm.plmn.length < 6 && gnbOperType != "view"' @click="commonAddOrUpdateParamClick('','add','plmn','')"><i class='el-icon el-icon-circle-add importIcon'></i> </div>
															</div>
															<div style="height: fit-content">
																<el-table ref="plmnTable" :rownumber="true" id="plmnTable" :data="gnbAddOrEditForm.plmn" :pagination="false" :row-key="'index'" default-expand-all style="border:1px solid #E9E9E9; min-height: 200px;">
																	<el-table-column type="expand">
																		<template slot-scope="props">
																			<div style='padding: 10px 0px;'>
																				<div style="padding-bottom: 5px; display: flex;width: 100%;">
																					<span class="commonSize14"><%=rb.getString("GNBPLMNSheZhi")%></span>
																					<span class="commonNotes12" style="margin-left: 5px;"><%=rb.getString("BuChaoGuo6Ge")%></span>
																					<!--row,operType,模块, 父级数据(plmn nr)-->
																					<span class="el-icon el-icon-circle-add" v-show='props.row.plmnConfigList.length < 6 && gnbOperType != "view"' @click="commonAddOrUpdateParamClick('','add','plmnConfig', props.row)" style="margin-left: 10px;"></span>
																				</div>
																				<div v-if='props.row.plmnConfigList'>
																					<el-table ref="plmnConfigTable" :rownumber="true" id="plmnConfigTable" :data="props.row.plmnConfigList" :pagination="false" default-expand-all style="border:1px solid #E9E9E9;">
																						<el-table-column type="expand">
																							<template slot-scope="props">
																								<div style='padding: 10px 0px;'>
																									<div style="padding-bottom: 5px; display: flex;width: 100%;">
																										<span class="commonSize14"><%=rb.getString("GNBQiePianLieBiao")%></span>
																										<span class="commonNotes12" style="margin-left: 5px;"><%=rb.getString("BuChaoGuo6Ge")%></span>
																										<span class="el-icon el-icon-circle-add" v-show=' props.row.sliceList.length < 6 && gnbOperType != "view"' @click="commonAddOrUpdateParamClick('','add','sliceConfig', props.row)" style="margin-left: 10px;"> </span>
																									</div>
																									<div v-if='props.row.sliceList'>
																										<el-table ref="sliceListTable" :rownumber="true" id="sliceListTable" :data="props.row.sliceList" :pagination="false" style="border:1px solid #E9E9E9;">
																											<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100" v-if='gnbOperType != "view"'>
																												<template slot-scope="scope">
																													<span class="el-icon el-icon-operation-edit" @click="commonAddOrUpdateParamClick(scope.row,'edit','sliceConfig',props.row)" style="margin-right:15px;"></span>
																													<span class="el-icon el-icon-operation-delete" @click="commonDelParamClick(scope.row, 'sliceConfig', props.row)" ></span>
																												</template>
																											</el-table-column>
																											<el-table-column label='ID' min-width="120" prop="index" show-overflow-tooltip></el-table-column>
																											<el-table-column label='SD' min-width="120" prop="sd" show-overflow-tooltip>
																												<template slot-scope="scope">
																													<div v-if="scope.row.sd === '1'"><%=rb.getString("GNBFeiKong")%></div>
																													<div v-else><%=rb.getString("GNBKong")%></div>
																												</template>
																											</el-table-column>
																											<el-table-column label='SNSSAI' min-width="120" prop="sd_value" show-overflow-tooltip></el-table-column>
																										</el-table>
																									</div>
																								</div>
																							</template>
																						</el-table-column>
																						<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100" v-if='gnbOperType != "view"'>
																							<template slot-scope="scope">
																								<span class="el-icon el-icon-operation-edit" @click="commonAddOrUpdateParamClick(scope.row,'edit','plmnConfig', props.row)" style="margin-right:15px;"></span>
																								<span class="el-icon el-icon-operation-delete" @click="commonDelParamClick(scope.row, 'plmnConfig', props.row)" ></span>
																							</template>
																						</el-table-column>
																						<el-table-column label='ID' prop="index" show-overflow-tooltip></el-table-column>
																						<el-table-column label='<%=rb.getString("GNBPLMNBiaoShi")%>' prop="plmnId" show-overflow-tooltip></el-table-column>
																						<el-table-column label='<%=rb.getString("GNBZhuPLMN")%>' prop="primary" show-overflow-tooltip></el-table-column>
																					</el-table>
																				</div>
																			</div>
																		</template>
																	</el-table-column>
																	<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100" v-if='gnbOperType != "view"'>
																		<template slot-scope="scope">
																			<span class="el-icon el-icon-operation-edit" @click="commonAddOrUpdateParamClick(scope.row,'edit','plmn', '')" style="margin-right:15px;"></span>
																			<span class="el-icon el-icon-operation-delete" @click="commonDelParamClick(scope.row, 'plmn', '')" ></span>
																		</template>
																	</el-table-column>
																	<el-table-column label='ID' min-width="120" prop="index" show-overflow-tooltip></el-table-column>
																	<el-table-column label='<%=rb.getString("GNBNCI")%>' min-width="120" prop="NCI" show-overflow-tooltip></el-table-column>
																	<el-table-column label='<%=rb.getString("GNBGenZongQuYuMa")%>' min-width="120" prop="tac" show-overflow-tooltip></el-table-column>
																	<el-table-column label='<%=rb.getString("GNBWuXianJieRuWangQuYuMa")%>' min-width="120" prop="ranac" show-overflow-tooltip></el-table-column>
																</el-table>
																<el-form-item prop='plmn' style="margin: 0 0 20px 0;" label-width="0"><el-input v-model='gnbAddOrEditForm.plmn' v-show="false"></el-input></el-form-item>
															</div>
														</div>
													</div>
												</el-collapse-item><!--Advance-->										
												<el-collapse-item name="advance">
													<template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonText14'><%=rb.getString("Advance")%></span></p></template>
													<div style='margin-left: 26px; margin-right: 0px;'>
														<div class='commonFlex commonContent' style="justify-content: flex-start;">
															<span class='commonGeneralBold12 ' style='padding: 6px 0;'><%=rb.getString("GNBCanShuPeiZhi")%></span>	
															<span class='commonTitle12' style='padding: 6px;'><%=rb.getString("GaoJingXuanZeCanShuPeiZhi")%></span>
															<span v-if="gnbOperType != 'view'" @click='gnbAdvanceSettingClick' class='licenseImportBox'><i class='el-icon el-icon-plus importIcon'></i></span>
														</div><!-- NR Setting AMF -->
														<div v-if="advanceTreeSelected.includes('AMF') && groupShow.AMFSetting">
															<div  class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='tieleTipCircle'></span>
																<span class='commonText14' style='padding: 6px 0;'>AMF</span>		
																<span v-if="gnbOperType != 'view'" @click="removeGroupItem('AMF')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>
															</div> 
															<div style='padding: 5px 0px 20px 18px;' class='paramPoolWarp'>
																<div class="contentTableTitle" style="padding-bottom: 5px;">
																	<div class="commonSize14"><%=rb.getString("AMFIPLieBiao")%></div>
																	<div v-if="gnbOperType != 'view'" class='licenseImportBox' @click="commonAddOrUpdateParamClick('', 'add', 'AMF', '')" style="margin: 0">
																		<i class='el-icon el-icon-plus importIcon'></i>
																	</div>
																</div>
																<el-ctable ref="amfTable" :rownumber="true" id="amfTable" :data="gnbAddOrEditForm.amf" height="200px" :pagination="false" style="border:1px solid #E9E9E9;">
																	<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100" v-if="gnbOperType != 'view'">
																		<template slot-scope="scope">
																			<span class="el-icon el-icon-operation-delete" @click="commonDelParamClick(scope.row, 'AMF')"></span>
																		</template>
																	</el-table-column>
																	<el-table-column label='ID' min-width="40" prop="index" show-overflow-tooltip></el-table-column>
																	<el-table-column label='AMF IP' min-width="120" prop="amfIp" show-overflow-tooltip></el-table-column>
																	<el-table-column label='<%=rb.getString("GNBPLMNBiaoShi")%>' min-width="120" prop="plmnId" show-overflow-tooltip></el-table-column>
																	<el-table-column label='<%=rb.getString("APNMoRen")%>' min-width="120" prop="default" show-overflow-tooltip></el-table-column>
																</el-ctable>
																<el-form-item prop='amf' style="display:none;" label="" label-width="0px"><el-input v-model='gnbAddOrEditForm.amf'></el-input></el-form-item>	
															</div>
														</div><!--NETWORK interfaceSetting   -->
														<div v-if="advanceTreeSelected.includes('WANLAN') && groupShow.NETWORK">
															<div  class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='tieleTipCircle'></span>
																<span class='commonText14' style='padding: 6px 0;'><%=rb.getString("JieKouSheZhi")%></span>		
																<span v-if="gnbOperType != 'view'" @click="removeGroupItem('WANLAN')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>
															</div>
															<div style='padding: 5px 0px 0px 18px;' class='paramPoolWarp'>
																<div class="contentTableTitle" style="padding-bottom: 5px;">
																	<div class="commonSize14"><%=rb.getString("WANLANlieBiao")%></div>
																	<div v-if="gnbOperType != 'view'" class='licenseImportBox' @click="commonAddOrUpdateParamClick('', 'add', 'wan', '')" style="margin: 0">
																		<i class='el-icon el-icon-plus importIcon'></i>
																	</div>
																</div>
																<div class="cellTableBoxCls" style="padding-bottom:20px;">
																	<el-ctable ref="wanListTable" :rownumber="true" id="wanListTable" :data="gnbAddOrEditForm.wan" height="200px" :pagination="false" style="border:1px solid #E9E9E9;">
																		<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100" v-if="gnbOperType != 'view'">
																			<template slot-scope="scope">
																				<span class="el-icon el-icon-operation-edit" @click="commonAddOrUpdateParamClick(scope.row, 'edit', 'wan', '')" style="margin-right:15px;"></span>
																				<span class="el-icon el-icon-operation-delete" @click="commonDelParamClick(scope.row, 'wan', '')" ></span>
																			</template>
																		</el-table-column>
																		<el-table-column label='<%=rb.getString("IPLeiXing")%>' prop="addressType" min-width="100" show-overflow-tooltip>
																			<template slot-scope="scope">
																				<div v-if="scope.row.addressType == 'DHCP'">DHCP</div>
																				<div v-if="scope.row.addressType == 'Static'">Static</div>
																				<div v-if="scope.row.addressType == 'DHCPv6'">IPv6 DHCP</div>
																				<div v-if="scope.row.addressType == 'Staticv6'">IPv6 Static</div>
																			</template>
																		</el-table-column>
																		<el-table-column label='<%=rb.getString("IPDiZhi")%>' prop="ipAddress" min-width="120" show-overflow-tooltip></el-table-column>
																		<el-table-column label='<%=rb.getString("QianZhuiChangDuZiWangYanMa")%>' prop="subnetMask" min-width="200" show-overflow-tooltip>
																			<template slot-scope="scope">
																				<span v-if="scope.row.addressType && scope.row.addressType == 'Static'">{{scope.row.subnetMask}}</span>
																				<span v-else-if="scope.row.addressType && scope.row.addressType == 'Staticv6'">{{scope.row.prefixLength}}</span>
																				<span v-else></span>
																			</template>
																		</el-table-column>
																		<el-table-column label='<%=rb.getString("WangGuan")%>' prop="gateway" min-width="120" show-overflow-tooltip></el-table-column>
																		<el-table-column label='<%=rb.getString("ChengZaiLeiXing")%>' prop="bearType" min-width="100" show-overflow-tooltip></el-table-column>
																		<el-table-column label='<%=rb.getString("VLANMingCheng")%>' min-width="100" prop="vlanName" show-overflow-tooltip></el-table-column>
																		<el-table-column label='VLAN ID' min-width="80" prop="vlanId" show-overflow-tooltip></el-table-column>
																	</el-ctable>
																	<el-form-item prop='wan' style="display:none;" label="" label-width="0px"><el-input v-model='gnbAddOrEditForm.wan'></el-input></el-form-item>
																</div>
															</div>
															<div style='padding: 5px 0px 0px 18px;' class='paramPoolWarp'>
																<div class="contentTableTitle" style="padding-bottom: 5px;">
																	<div class="commonSize14"><%=rb.getString("LANLieBiao")%></div>
																	<div v-if="gnbOperType != 'view'" class='licenseImportBox' @click="commonAddOrUpdateParamClick('', 'add', 'lan', '')" style="margin: 0">
																		<i class='el-icon el-icon-plus importIcon'></i>
																	</div>
																</div>
																<div class="cellTableBoxCls" style="padding-bottom:20px;">
																	<el-ctable ref="lanListTable" :rownumber="true" id="lanListTable" :data="gnbAddOrEditForm.lan" height="200px" :pagination="false" style="border:1px solid #E9E9E9;">
																		<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100" v-if="gnbOperType != 'view'">
																			<template slot-scope="scope">
																				<span class="el-icon el-icon-operation-edit" @click="commonAddOrUpdateParamClick(scope.row, 'edit', 'lan', '')" style="margin-right:15px;"></span>
																				<span class="el-icon el-icon-operation-delete" @click="commonDelParamClick(scope.row, 'lan', '')" ></span>
																			</template>
																		</el-table-column>
																		<el-table-column label='ID' min-width="120" prop="index" show-overflow-tooltip></el-table-column>
																		<el-table-column label='<%=rb.getString("IPDiZhi")%>' min-width="120" prop="ipAddress" show-overflow-tooltip></el-table-column>
																		<el-table-column label='<%=rb.getString("GNBZiWangYanMa")%>' min-width="120" prop="subnetMask" show-overflow-tooltip></el-table-column>
																	</el-ctable>
																	<el-form-item prop='lan' style="display:none;" label="" label-width="0px">
																		<el-input v-model='gnbAddOrEditForm.lan'></el-input>
																	</el-form-item>
																</div>
															</div>
														</div><!--Network ipsecSetting-->
														<div v-if="advanceTreeSelected.includes('IPSEC') && groupShow.NETWORK">
															<div class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='tieleTipCircle'></span>
																<span class='commonText14' style='padding: 6px 0;'>IPSec</span>		
																<span v-if="gnbOperType != 'view'" @click="removeGroupItem('IPSEC')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>
															</div>
															<div style='padding: 5px 0px 20px 18px;' class='paramPoolWarpThree ipsecItem'>
																<el-form-item label='Enable' prop="ipsecEnable">
																	<el-switch onclick="event.stopPropagation()" v-model="gnbAddOrEditForm.ipsecEnable" active-value="1" inactive-value="0" :disabled="gnbIsReadOnly">
																	</el-switch>
																</el-form-item>															
																<el-form-item label="IMSI" prop="ipsecImsi" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.ipsecImsi" maxlength="1024" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("GNBChangDuTiShi")%>0-1024,<%=rb.getString("GNBZiFuChuan")%></template>
																	</el-input>
																</el-form-item>
																<el-form-item label="Key" prop="ipsecKey" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.ipsecKey" maxlength="1024" :disabled="gnbIsReadOnly">
																		<template slot="append"> <p><%=rb.getString("FanWei")%>:0-1024 <%=rb.getString("GNBCanShuJiaoYan")%></p></template>
																	</el-input>
																</el-form-item>
																<el-form-item label="OPC" prop="ipsecOpc" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.ipsecOpc" maxlength="1024" :disabled="gnbIsReadOnly">
																		<template slot="append"><p><%=rb.getString("FanWei")%>:0-1024 <%=rb.getString("GNBCanShuJiaoYan")%></p></template>
																	</el-input>
																</el-form-item>
																<el-form-item label='Usim Enable' prop="ipsecUsimEnable">
																	<el-select v-model='gnbAddOrEditForm.ipsecUsimEnable' :disabled="gnbIsReadOnly">
																		<el-option label='Disable' value='0'></el-option>
																		<el-option label="Enable" value='1'></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label='IpsecUsimAuthenticationEnable' prop="ipsecUsimAuthEnable">
																	<el-select v-model='gnbAddOrEditForm.ipsecUsimAuthEnable' :disabled="gnbIsReadOnly">
																		<el-option label='Unbound' value='0'></el-option>
																		<el-option label='SN band' value='1'></el-option>
																		<el-option label='MAC band' value='2'></el-option>
																	</el-select>
																</el-form-item>
																<div class='paramPoolWarp'>
																	<div class="contentTableTitle" style="padding-bottom: 5px;">
																		<div >
																			<span class="commonSize14">IPSec Tunnel List</span>
																			<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("ZuiDuoSanGe")%>)</span>
																		</div>
																		<div v-if="addIPSecTunnelBtnShow && gnbOperType != 'view'" class='licenseImportBox' @click="addIPSecTunnelDialogOpen('','add')" style="margin: 0">
																			<i class='el-icon el-icon-plus importIcon'></i>
																		</div>
																	</div>
																	<el-ctable ref="ipsecTunnelTable" :rownumber="true" id="ipsecTunnelTable" :data="gnbAddOrEditForm.ipsecTunnel" height="145px" :pagination="false" style="border:1px solid #E9E9E9;">
																		<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100" v-if="gnbOperType != 'view'">
																			<template slot-scope="scope">
																				<span class="el-icon el-icon-operation-edit" @click="addIPSecTunnelDialogOpen(scope.row,'edit')" style="margin-right:15px;"></span>
																				<span class="el-icon el-icon-operation-delete" @click="delIPSecTunnelList(scope.row,'IPSec',event)"></span>
																			</template>
																		</el-table-column>
																		<el-table-column label='<%=rb.getString("IPSECGuanDaoMingChen")%>' min-width="120" prop="tunnel_name" show-overflow-tooltip></el-table-column>
																		<el-table-column label='<%=rb.getString("SheZhiKaiGuan")%>' min-width="100" prop="tunnel_enable" show-overflow-tooltip>
																			<template slot-scope="scope">
																				<div v-if="scope.row.tunnel_enable == '0'"><span><%=rb.getString("Guan")%></span></div>
																				<div v-if="scope.row.tunnel_enable == '1'"><span>><%=rb.getString("Kai")%></span></div>
																			</template>
																		</el-table-column>
																		<el-table-column label='<%=rb.getString("ZhuangTai")%>' min-width="120" prop="status" show-overflow-tooltip>
																			<template slot-scope="scope">
																				<div v-if="scope.row.status != '2'"><span><%=rb.getString("IpsecWeiLianJie")%></span></div>
																				<div v-if="scope.row.status == '2'"><span><%=rb.getString("LianJieZhengChang")%></span></div>
																			</template>
																		</el-table-column>
																		<el-table-column label='<%=rb.getString("IPDiZhi")%>' min-width="120" prop="used_source_ip" show-overflow-tooltip></el-table-column>
																		<el-table-column label='<%=rb.getString("WangGuan")%>' min-width="120" prop="gateway" show-overflow-tooltip></el-table-column>
																	</el-ctable>
																	<el-form-item prop='ipsecTunnel' style="display:none;" label="" label-width="0px">
																		<el-input v-model='gnbAddOrEditForm.ipsecTunnel'></el-input>
																	</el-form-item>
																</div>
															</div>
														</div><!--Network staticRoutingSetting-->
														<div v-if="advanceTreeSelected.includes('STATICROUTE') && groupShow.NETWORK">
															<div  class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='tieleTipCircle'></span>
																<span class='commonText14' style='padding: 6px 0;'><%=rb.getString("LuYouSheZhi")%></span>		
																<span v-if="gnbOperType != 'view'" @click="removeGroupItem('STATICROUTE')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>
															</div>
															<div style='padding: 5px 0px 20px 18px;' class='paramPoolWarp'>
																<div class="contentTableTitle" style="padding-bottom: 5px;">
																	<div class="commonSize14"><%=rb.getString("LuYouLieBiao")%></div>
																	<div v-if="gnbOperType != 'view'" class='licenseImportBox' @click="commonAddOrUpdateParamClick('', 'add', 'router', '')" style="margin: 0">
																		<i class='el-icon el-icon-plus importIcon'></i>
																	</div>
																</div>
																<el-ctable ref="staticListTable" :rownumber="true" id="staticListTable" :data="gnbAddOrEditForm.staticRouting" height="200px" :pagination="false" style="border:1px solid #E9E9E9;">
																	<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100" v-if="gnbOperType != 'view'">
																		<template slot-scope="scope">
																			<span class="el-icon el-icon-operation-edit" @click="commonAddOrUpdateParamClick(scope.row, 'edit', 'router', '')" style="margin-right:15px;"></span>
																			<span class="el-icon el-icon-operation-delete" @click="commonDelParamClick(scope.row, 'router', '')" ></span>
																		</template>
																	</el-table-column>
																	<el-table-column label='<%=rb.getString("JieKouMingCheng")%>' prop="interfaceName" v-if='false' show-overflow-tooltip></el-table-column>
																	<el-table-column label='<%=rb.getString("MuDiWangLuo")%>' prop="destinationNetwork" show-overflow-tooltip></el-table-column>
																	<el-table-column label='<%=rb.getString("ZiWangYanMa")%>' prop="netmask" show-overflow-tooltip></el-table-column>
																	<el-table-column label='<%=rb.getString("WangGuan")%>' prop="gateway" show-overflow-tooltip></el-table-column>
																</el-ctable>
																<el-form-item prop='staticRouting' style="display:none;" label="" label-width="0px">
																	<el-input v-model='gnbAddOrEditForm.staticRouting'></el-input>
																</el-form-item>
															</div>
														</div><!--HaloB-->
														<div v-if="advanceTreeSelected.includes('HALOB') && groupShow.NETWORK">
															<div  class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='tieleTipCircle'></span>
																<span class='commonText14' style='padding: 6px 0;'>HaloB</span>		
																<span v-if="gnbOperType != 'view'" @click="removeGroupItem('HALOB')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>
															</div>
															<div style='padding: 5px 0px 0 18px;' class='paramPoolWarp'>
																<el-form-item label='<%=rb.getString("HaloBKaiGuan")%>' prop="halobEnable">
																	<el-switch onclick="event.stopPropagation()" v-model="gnbAddOrEditForm.halobEnable" active-value="1" inactive-value="0" :disabled="gnbIsReadOnly">
																	</el-switch>
																</el-form-item>
																<el-form-item label='<%=rb.getString("GNBHaloBMoShi")%>' prop="halobMode">
																	<el-select v-model='gnbAddOrEditForm.halobMode' :disabled="gnbIsReadOnly">
																		<el-option label="Centralized" value="1"></el-option>
																		<el-option label="Single" value="2"></el-option>
																	</el-select>
																</el-form-item>
															</div>
														</div><!--Management Server-->
														<div v-if="advanceTreeSelected.includes('MANAGEMENT') && groupShow.BTSSetting">
															<div  class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='tieleTipCircle'></span>
																<span class='commonText14' style='padding: 6px 0;'><%=rb.getString("GuanLiFuWu")%></span>		
																<span v-if="gnbOperType != 'view'" @click="removeGroupItem('MANAGEMENT')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>
															</div>
															<div style='padding: 5px 0px 0 18px;' class='paramPoolWarpThree'>
																<el-form-item prop="periodicInformEnable">
																	<div slot="label">
																		<span class="commonNotes12" style="margin-left: 5px;color: #f56c6c;">*</span>
																		<span class="commonSize14"><%=rb.getString("DingQiTongZhiKaiGuan")%></span>
																	</div>
																	<el-switch onclick="event.stopPropagation()" v-model="gnbAddOrEditForm.periodicInformEnable" active-value="1" inactive-value="0" :disabled="gnbIsReadOnly"></el-switch>
																</el-form-item>
																<el-form-item label='<%=rb.getString("DingQiTongZhiShiJian")%>' prop="periodicInformTime" class='inputCommon'>
																	<el-date-picker style='width: 200px;' v-model="gnbAddOrEditForm.periodicInformTime" :disabled="gnbIsReadOnly" type="datetime" value-format="yyyy-MM-dd HH:mm:ss">
																	</el-date-picker>
																</el-form-item>
																<el-form-item label='<%=rb.getString("DingQiTongZhiJianGe")%>' prop="periodicInformInterval" class='inputCommon validateItem'>
																	<el-input v-model.trim="gnbAddOrEditForm.periodicInformInterval" :disabled="gnbIsReadOnly">
																		<template slot="append"><%=rb.getString("FanWei")%>：<%=rb.getString("GNBWuXian")%>,<%=rb.getString("ZhengXing")%></template>
																	</el-input>
																</el-form-item>
															</div>
														</div><!--NTP-->
														<div v-if="advanceTreeSelected.includes('NTP') && groupShow.SYSTEM">
															<div  class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='tieleTipCircle'></span>
																<span class='commonText14' style='padding: 6px 0;'>NTP</span>		
																<span v-if="gnbOperType != 'view'" @click="removeGroupItem('NTP')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>
															</div>
															<div style='padding: 5px 0px 0 18px;' class='paramPoolWarpThree'>
																<el-form-item prop="ntpEnable">
																	<div slot="label">
																		<span class="commonNotes12" style="margin-left: 5px;color: #f56c6c;">*</span>
																		<span class="commonSize14"><%=rb.getString("NTPKaiGuan")%></span>
																	</div>
																	<el-switch onclick="event.stopPropagation()" v-model="gnbAddOrEditForm.ntpEnable" active-value="1" inactive-value="0" :disabled="gnbIsReadOnly"></el-switch>
																</el-form-item>
																<el-form-item label='<%=rb.getString("GNBShiQu")%>' prop="localTimeZone" class='inputCommon'>
																	<el-select v-model='gnbAddOrEditForm.localTimeZone' filterable :disabled="gnbIsReadOnly">
																		<el-option v-for="item in timeZoneList" :key="item.value" :label="item.label" :value="item.value"></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label='<%=rb.getString("NTPFuWuQi")%> 1' prop="ntpServer1" class='inputCommon'>
																	<el-input v-model.trim="gnbAddOrEditForm.ntpServer1" :disabled="gnbIsReadOnly"></el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("NTPFuWuQi")%> 2' prop="ntpServer2" class='inputCommon'>
																	<el-input v-model.trim="gnbAddOrEditForm.ntpServer2" :disabled="gnbIsReadOnly"></el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("NTPFuWuQi")%> 3' prop="ntpServer3" class='inputCommon'>
																	<el-input v-model.trim="gnbAddOrEditForm.ntpServer3" :disabled="gnbIsReadOnly"></el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("NTPFuWuQi")%> 4' prop="ntpServer4" class='inputCommon'>
																	<el-input v-model.trim="gnbAddOrEditForm.ntpServer4" :disabled="gnbIsReadOnly"></el-input>
																</el-form-item>
																<el-form-item label='<%=rb.getString("NTPFuWuQi")%> 5' prop="ntpServer5" class='inputCommon'>
																	<el-input v-model.trim="gnbAddOrEditForm.ntpServer5" :disabled="gnbIsReadOnly"></el-input>
																</el-form-item>
															</div>
														</div>
													</div>
												</el-collapse-item><!--Customized Paeameters-->
												<el-collapse-item name="customized">
													<template slot='title'>
														<div style="display: flex;margin-left: 26px;">
															<span class='commonText14'><%=rb.getString("ZiDingYiCanShu")%></span>
															<div v-if="gnbOperType != 'view'" class='licenseImportBox' @click="addNrPlmnAddClick" style="margin: 10px 10px 0">
																<i class='el-icon el-icon-plus importIcon'></i>
															</div>
														</div>
													</template>
													<div style='margin-left: 26px; margin-right: 0px;' v-show="gnbAddOrEditForm.custParam.length > 0">
														<div style='width: 100%;' class='paramPoolWarpThree'>
															<div v-for="(item,index) in gnbAddOrEditForm.custParam" class="specialItemCls">
																<div style="display:flex;flex-wrap: wrap;font-size:12px">
																	<el-form-item label="ID" :prop="'custParam['+index+'].id'" class='validate-item' v-if="false"> 
																		<el-input v-model.trim='item.id' :disabled="gnbIsReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label='<%=rb.getString("MingCheng")%>' :prop="'custParam['+index+'].custParamName'" class='validate-item'> 
																		<el-input v-model.trim='item.custParamName' :disabled="gnbIsReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label='<%=rb.getString("GNBZhi")%>' :prop="'custParam['+index+'].custParamValue'" class='validate-item'> 
																		<el-input v-model.trim='item.custParamValue' :disabled="gnbIsReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label='<%=rb.getString("GNBLuJing")%>' :prop="'custParam['+index+'].custParamPath'"  class='validate-item' > 
																		<el-input v-model.trim='item.custParamPath' :disabled="gnbIsReadOnly" style="min-width: 250px;"></el-input>
																	</el-form-item>
																</div>															
																<div class="specialItemDelIcon" v-if="gnbOperType != 'view'">
																	<span class="el-icon el-icon-circle-close" @click="addNrPlmnListDel(item)" style="font-size: 12px;"></span>
																</div>
															</div>
															<el-form-item prop='custParam' style="display:none;" label-width="0px">
																<el-input v-model='gnbAddOrEditForm.custParam'></el-input>
															</el-form-item>
														</div>
													</div>
												</el-collapse-item>
											</el-collapse>
										</div>
                                    </el-tab-pane><!--batch import-->
                                    <el-tab-pane label='<%=rb.getString("ZhiDingSheBeiJiHua")%>' name="gnbParamsImport">
                                        <div class='paramTwoBox'>
                                            <el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='batchConfigEnable' label-width='80px' class="enableCommon basicLeftLabel" style='margin-bottom: 16px;'>
                								<el-switch v-model="gnbAddOrEditForm.batchConfigEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="gnbIsReadOnly"></el-switch>
                                            </el-form-item>
											<div style='height: 300px;'>
												<el-ctable ref="paramsImportFileTable" id='paramsImportFileTable' :url="urlConfigPlan" :time="6" :query-params="importParamsQuery" :row-key="'id'" class='commonBorderRadius commonBorder' style='margin-top: 10px; margin-bottom: 10px;'
													:height="height" pagination="true" rownumber="true">													
													<template slot="toolbar">
														<div class='commonFlex commonToolBarBox commonContent licenseBox'>
															<span class='commonText14' style='padding: 6px 20px;'><%=rb.getString("ZhiDingCanShuLieBiao")%></span>
															<div class='commonFlex'>
																<div class="queryGroup curImportQuery" style='margin-right: 6px;'>
																	<el-input v-model="importParamsQueryForm.searchText" @keyup.enter.native="queryPlan" class='pairgrid-query' style='width: 220px;' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
																	<i @click="queryPlan" class="el-icon el-icon-common-search commonLeft10"></i>
																</div>
																<div v-show='gnbOperType !="view"' class='licenseImportBox' @click='gnbParamImportFileClick'><i class='el-icon el-icon-operation-import importIcon'></i></div>
																<div v-if='false' class='licenseImportBox' @click='gnbParamExportClick' style='margin-left: 0 !important;'><i class='el-icon el-icon-operation-export importIcon'></i></div>
															</div>
														</div>
													</template>
													<el-table-column width="100">
														<template slot-scope="scope">
															<div>
																<i v-show='gnbOperType != "view"' @click="paramsImportTableEditClick(scope.row, event)" class="el-icon el-icon-operation-edit commonIcon"></i>
																<i v-show='gnbOperType != "view"' @click="paramsImportTableDelClick(scope.row, event)" class="el-icon el-icon-operation-delete commonIcon" style='margin-left: 6px;'></i>
																<i @click="paramsImportTableViewClick(scope.row, event)" class="el-icon el-icon-operation-info commonIcon" style='margin-left: 6px;'></i>
															</div>
														</template>
													</el-table-column>
													<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label='<%=rb.getString("GNBMingCheng")%>' prop="gnb_name" width="230" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label='<%=rb.getString("GNBBiaoShiChangDu")%>' prop="gnb_length" width="230" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label='<%=rb.getString("GNBBiaoShi")%>' prop="gnb_id" width="160" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label='PCI' prop="pci" width="160" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label='User' prop="user" width="160" show-overflow-tooltip="true" v-if='false'></el-table-column>
													<el-table-column label='Update Time' prop="update_time" width="200" show-overflow-tooltip="true" v-if='false'></el-table-column>
												</el-ctable>												
								            </div>
                                        </div>
                                    </el-tab-pane>
                                </el-tabs>
                            </div>
			            </div>
			        </div>
                </el-form>
   			</div>
		    <div class='commonFlex commonContent footerBox commonBorderTop' v-show="gnbOperType !='view'">
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" size="mini" @click='gnbAddOrUpdateSubmit' :disabled='gnbSaveBtnDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='gnbAddOrUpdateCancel'><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div><!--software upgrade: Add Version  -->
   		<div class='rightBox' style='position: relative;' v-show='gnbAddVersionShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='slideTopTitle commonText14'><%=rb.getString("TianJianBanBen")%></span>
   				<span class='closeIconBox' @click='gnbAddVersionCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 10px 0 20px;'>
   				<div class='rightTextBox'><span class='commonTitle12'><%=rb.getString("ShouDongHuoXuanZeBanBen")%></span></div>
   				<div class='originalVersionBox'>
   				   	<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 6px; '><%=rb.getString("ChuShiBanBen") %></span>
   					<el-form ref="gnbSoftwareAddVersionForm" :inline="true" label-position="top" :model="gnbSoftwareAddVersionForm" style="display: flex; flex-direction: column;overflow: hidden;">
	   					<el-form-item style='margin-bottom: 16px;'>
							<el-form-item prop='versionStr'>
								<el-input v-model='gnbSoftwareAddVersionForm.versionStr'><i slot="append" class="el-icon el-icon-plus" @click="gnbAddVersionBtn"></i></el-input>
							</el-form-item>
							<div class='versionResultBox' v-show='gnbSoftwareAddVersionForm.versionList.length > 0'>
								<el-form-item class='suffixItem' v-for='(domain,index) in gnbSoftwareAddVersionForm.versionList'>
									<div class='form-suffix'>
										<span class='text'>{{domain}}</span>
										<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteVersion' @click.prevent='gnbRemoveVersion(domain)'></span>
									</div>
								</el-form-item>
								<el-form-item prop='itemTest'>
									<el-input v-model='gnbSoftwareAddVersionForm.itemTest' v-show=false></el-input>
								</el-form-item>
							</div>
							<p class='ipErrorTip'>{{gnbVersionErrorMessage}}</p>
	                    </el-form-item>
   					</el-form>
   				</div>
   				<div>
	                <span class='commonTitle12 commonDisplayBlock'><%=rb.getString("ChuShiBanBenLieBiao")%></span>
					<el-ctable ref="gnbSoftwareOriginalVersionTable" height='280px' class='commonBorderRadius commonBorder' style='margin-top: 8px;' :pagination="false" :rownumber="false"
						:data="gnbSoftwareOriginalVersionList" row-key="originalVersion" @selection-change='gnbVersionBatchSelect'>
						<el-table-column type="selection" :reserve-selection="true"></el-table-column>
						<el-table-column label='<%=rb.getString("ChuShiBanBen") %>' prop="originalVersion" show-overflow-tooltip="true"></el-table-column>
					</el-ctable>
	            </div>
	            <p class='ipErrorTip' style='margin-top: 2px;'>{{gnbVersionMessage}}</p>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" size="mini" @click='gnbAddVersionSubmit'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='gnbAddVersionCancel'><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div><!--import license  -->
   		<div class='rightBox' style='position: relative;' v-show='gnbLicenseImportShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("DaoRu")%> License</span>
   				<span class='closeIconBox' @click='gnbLicenseImportCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='rightTextBox'><span class='commonTitle12'><%=rb.getString("DaoRuLicenseTiShi")%></span></div>
   				<div class='originalVersionBox'>
					<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 6px; '><%=rb.getString("WenJianMing")%>(<span class='commonTitle12'><%=rb.getString("LicenseGeShi")%></span>)</span>
   					<el-form label-position="top" ref="gnbLicenseForm" :model='gnbLicenseForm' :rules='gnbImportLicenseRules' v-loading="gnbLicenseLoading">
	                  	<el-form-item label=" " prop="fileName">
							<el-upload ref="moreUpload" :multiple="true" :on-success='gnbMoreCheckFile' :on-change="gnbMoreFileChange" :show-file-list=false :action="gnbLicenseForm.uploadFileUrl" 
								:data="gnbMoreFileParams" name="uploadFile" :file-list="gnbMoreFileList" :http-request="gnbMoreFileRequest" :auto-upload="false" accept=".lic"> 
								<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
									<a slot="append" class="el-icon el-icon-operation-import importBox" @click="gnbMoreFileSelect"></a>
								</el-input>
								<a slot="trigger" ref="more_file_up"></a>
							</el-upload>
	                    </el-form-item>
		            </el-form>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="gnbLicenseImportSubmit" :disabled='gnbUploadBtnDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button @click="gnbLicenseImportCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div><!-- Advance Setting Tree -->
		<div class='rightBox320' style='position: relative;' v-show='gnbAdvanceTreeShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 10px 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("TianJia")%></span>
   				<span class='closeIconBox' @click='gnbAdvanceSettingCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'><%=rb.getString("XuanZeCanShuLaiPeiZhi")%></span></div>
   				<div class='originalVersionBox'>   					
					<el-tree show-checkbox ref="advanceTree" node-key="id" :data="advanceTree" :check-on-click-node="true" :default-expanded-keys="['AMFSetting','NETWORK','BTSSetting','SYSTEM']" :default-checked-keys="defaultRreeChecked">
					</el-tree>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop' style='width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100;'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="gnbAdvanceTreeConfirm"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="gnbAdvanceSettingCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div><!--参数配置模块：新建，修改表格操作： 右侧操作窗口 -->
		<div class='rightBox' style='position: relative;' v-show="commonAddOrUpdateParamShow">
			<div class='commonRightTitleBox'>
   				<span class='AddTitle commonText14'>{{commonAddOrUpdateParamTitle}}</span>
   				<span class='el-icon el-icon-close' @click='commonAddOrUpdateParamCancel' style="font-size: 12px; margin: 12px 20px;"></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='originalVersionBox' >
					<el-form ref="commonAddEditForm" :model='commonAddEditForm' :rules='commonAddEditFormRules' label-position="top">     		     			            
						<div v-if="commonAdvanceOperModel == 'AMF'">
							<el-form-item prop='amfIp' class="titleAndNoteBox">
								<div slot="label">
									<span class="commonSize14">AMF IP</span>
									<span class="commonNotes12" style="margin-left: 5px;">(Example：1.1.1.1)</span>
								</div>
								<el-input v-model.trim='commonAddEditForm.amfIp'></el-input>
							</el-form-item>
							<el-form-item label='<%=rb.getString("GNBPLMNBiaoShi")%>' prop='plmnId' class="titleAndNoteBox">
								<el-select v-model='commonAddEditForm.plmnId' >
									<el-option v-for="item in plmnIdList" :label='item' :value='item'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item label='<%=rb.getString("APNMoRen")%>' prop='default'>
								<el-select v-model='commonAddEditForm.default'>
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
								</el-select>
							</el-form-item>
						</div>
						<div v-if="commonAdvanceOperModel == 'router'">
							<el-form-item label='<%=rb.getString("MuDiWangLuo")%>' prop='destinationNetwork'>
								<el-input v-model.trim='commonAddEditForm.destinationNetwork'></el-input>
							</el-form-item>
							<el-form-item label='<%=rb.getString("GNBZiWangYanMa")%>' prop='netmask'>
								<el-input v-model.trim='commonAddEditForm.netmask'></el-input>
							</el-form-item>
							<el-form-item label='<%=rb.getString("WangGuan")%>' prop='gateway'>
								<el-input v-model.trim='commonAddEditForm.gateway'></el-input>
							</el-form-item>
						</div>
						<div v-if="commonAdvanceOperModel == 'wan'">
							<el-form-item label='<%=rb.getString("IPLeiXing")%>' prop='addressType'>
								<el-select v-model='commonAddEditForm.addressType'>
									<el-option label='DHCP' value='DHCP'></el-option>
									<el-option label='Static' value='Static'></el-option>
									<el-option label='IPv6 DHCP' value='DHCPv6'></el-option>
									<el-option label='IPv6 Static' value='Staticv6'></el-option>
								</el-select>
							</el-form-item> 
							<el-form-item label='<%=rb.getString("ChengZaiLeiXing")%>' prop='bearType'>
								<el-select v-model='commonAddEditForm.bearType'>
									<el-option label='NG' value='NG'></el-option>
									<el-option label='OAM' value='OAM'></el-option>
									<el-option label='NGU' value='NGU'></el-option>
									<el-option label='NG/OAM' value='NG/OAM'></el-option>
									<el-option label='NG/NGU' value='NG/NGU'></el-option>
									<el-option label='NGU/OAM' value='NGU/OAM'></el-option>
									<el-option label='NG/OAM/NGU' value='NG/OAM/NGU'></el-option>
								</el-select>
							</el-form-item> 
							<el-form-item label="IP" v-if="['Static','Staticv6'].includes(commonAddEditForm.addressType)" prop='ipAddress'>
								<el-input v-model.trim='commonAddEditForm.ipAddress'></el-input>
							</el-form-item>
							<el-form-item label='<%=rb.getString("GNBZiWangYanMa")%>' v-if="['Static'].includes(commonAddEditForm.addressType )" prop='subnetMask'>
								<el-input v-model.trim='commonAddEditForm.subnetMask'></el-input>
							</el-form-item>
							<el-form-item v-if="['Staticv6'].includes(commonAddEditForm.addressType)" prop='prefixLength' class="titleAndNoteBox">
								<div slot="label">
									<span class="commonSize14"><%=rb.getString("QianZhuiChangDu")%></span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：0~128,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='commonAddEditForm.prefixLength'></el-input>
							</el-form-item>
							<el-form-item label='<%=rb.getString("WangGuan")%>' v-if="['Static','Staticv6'].includes(commonAddEditForm.addressType)" prop='gateway'>
								<el-input v-model.trim='commonAddEditForm.gateway' ></el-input>
							</el-form-item>
							<el-form-item label="Origin" prop='origin' v-if='false'>
								<el-input v-model.trim='commonAddEditForm.origin'></el-input>
							</el-form-item>
							<div v-if="!addVlanShow">
								<el-form-item label="VLAN ID" prop='vlanId' style="margin-bottom: 10px; ">
									<el-select v-model='commonAddEditForm.vlanId' >
										<el-option v-for="item in vlanIdList" :label='item' :value='item'></el-option>
									</el-select>
								</el-form-item> 
								<div class='commonFlex' style="margin-top: 4px;">
									<span class='commonImportSize14'><%=rb.getString("TianJiaXinDeVlan")%></span>
									<div class="allowMoreInputAddBtnCls" @click="addVlanClick('params')">
										<span class="el-icon el-icon-plus"></span>
										<span><%=rb.getString("TianJia")%></span>
									</div>
								</div>
							</div>
							<div v-if="addVlanShow">
								<span class="commonSize14"><%=rb.getString("TianJiaVlan")%></span>
								<div class="specialItemCls" style="margin: 10px 0;padding: 20px 20px 0;">
									<div>
										<el-form-item prop="vlanName" class="titleAndNoteBox">
											<div slot="label">
												<span class="commonSize14"><%=rb.getString("VLANMingCheng")%></span>
												<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("GNBChangDuTiShi")%>1~13,<%=rb.getString("ZiFuFuShu")%>)</span>
											</div>
											<el-input v-model.trim='commonAddEditForm.vlanName' maxlength="13" style="width: 270px;"></el-input>
										</el-form-item>
										<el-form-item prop="vlanId" class="titleAndNoteBox">
											<div slot="label">
												<span class="commonSize14">VLAN ID</span>
												<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：2~4094,<%=rb.getString("ZhengXing")%>)</span>
											</div>
											<el-input v-model.trim='commonAddEditForm.vlanId' style="width: 270px;"></el-input>
										</el-form-item>
									</div>
									<div class="specialItemDelIcon">
										<span class="el-icon el-icon-circle-close" @click="closeAddVlanClick('params')"></span>
									</div>
								</div>
							</div>
						</div>
						<div v-if="commonAdvanceOperModel == 'lan'">
							<el-form-item label='<%=rb.getString("IPLeiXing")%>' prop='addressType' v-if='false'> 
								<el-select v-model='commonAddEditForm.addressType'><el-option label='Static' value='Static'></el-option></el-select>
							</el-form-item> 
							<el-form-item label='<%=rb.getString("IPDiZhi")%>' prop='ipAddress'>
								<el-input v-model.trim='commonAddEditForm.ipAddress'></el-input>
							</el-form-item>
							<el-form-item label='<%=rb.getString("GNBZiWangYanMa")%>' prop='subnetMask'>
								<el-input v-model.trim='commonAddEditForm.subnetMask'></el-input>
							</el-form-item>
						</div>
						<div v-if="commonAdvanceOperModel == 'plmn'">
							<p class='commonText14' style="padding-bottom: 20px;"><%=rb.getString("PLMNBiaoShiXinXiSheZhi")%></p>
							<el-form-item prop='NCI' class="titleAndNoteBox">
								<div slot="label">
									<span class="commonSize14"><%=rb.getString("GNBNCI")%></span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：0~68719476735,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='commonAddEditForm.NCI'></el-input>
							</el-form-item> 
							<el-form-item prop='tac' class="titleAndNoteBox">
								<div slot="label">
									<span class="commonSize14"><%=rb.getString("GNBGenZongQuYuMa")%></span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：0~16777215,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='commonAddEditForm.tac'></el-input>
							</el-form-item>
							<el-form-item prop='ranac' class="titleAndNoteBox">
								<div slot="label">
									<span class="commonSize14"><%=rb.getString("GNBWuXianJieRuWangQuYuMa")%></span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：0~255,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='commonAddEditForm.ranac'></el-input>
							</el-form-item>
						</div>
						<div v-if="commonAdvanceOperModel == 'plmnConfig'">
							<p class='commonText14' style="padding-bottom: 20px;"><%=rb.getString("GNBPLMNSheZhi")%></p>
							<el-form-item prop='plmnId' class="titleAndNoteBox">
								<div slot="label">
									<span class="commonSize14"><%=rb.getString("GNBPLMNBiaoShi")%></span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：5~6 Digit,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='commonAddEditForm.plmnId'></el-input>
							</el-form-item>
							<el-form-item label='<%=rb.getString("GNBZhuPLMN")%>' prop='primary'>
								<el-select v-model='commonAddEditForm.primary'>
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
								</el-select>
							</el-form-item>
						</div>
						<div v-if="commonAdvanceOperModel == 'sliceConfig'">
							<p class='commonText14' style="padding-bottom: 20px;">Slice Setting</p>
							<el-form-item label="SD" prop='sd'>
								<el-radio-group v-model="commonAddEditForm.sd">
									<el-radio label="0" border><%=rb.getString("GNBKong")%></el-radio>
									<el-radio label="1" border><%=rb.getString("GNBFeiKong")%></el-radio>
								</el-radio-group>
							</el-form-item>
							<el-form-item prop='sd_value' v-if='commonAddEditForm.sd == "1"' class="titleAndNoteBox">
								<div slot="label">
									<span class="commonSize14">SNSSAI</span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：0~16777215,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='commonAddEditForm.sd_value'></el-input>
							</el-form-item>
						</div>
					</el-form> 
   				</div>
			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="commonAddOrUpdateParamSubmit" :disabled='commonAddOrUpdateBtnDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button @click="commonAddOrUpdateParamCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div><!-- IPSecTunnel add, edit 弹窗 -->
		<el-dialog class="gnbIpsecConfigAddDialog" top="25vh" :title="gnbIpsecConfigAddDialogTitle" width="54%" :visible.sync="addIPSecTunnelDialogShow" @close="closeAddIPSecTunnelDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
			<div class="gnbConfigAddMainBoxCls">
				<el-form ref="addIPSecDialogForm" :model='addIPSecDialogForm' :rules='addIPSecDialogRules' label-position="top">     		     			            
					<el-collapse v-model="activeIPSecTunnelCollapse">
						<el-collapse-item name="Basic">
							<template slot='title'><p style="display:inline-block;margin-left:40px;"><span style="font-size:14px;font-weight:bold"><%=rb.getString("JiBenSheZhi")%></span></p></template>
							<div class="rightContentCls" >
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='tunnel_enable' style="width:40%;min-width:400px;" label='<%=rb.getString("SheZhiKaiGuan")%>' label-width="160px">
										<el-switch v-model="addIPSecDialogForm.tunnel_enable" active-value="1" inactive-value="0" style='padding-top:10px;'></el-switch>
									</el-form-item>
									<el-form-item prop='tunnel_name' style="width:40%;min-width:400px;" label='<%=rb.getString("IPSECGuanDaoMingChen")%>' label-width="160px" class='validate-item'>
										<el-input v-model.trim='addIPSecDialogForm.tunnel_name' maxlength="64" :disabled="advanceIpsecOperType == 'edit'">
											<template slot="append"><%=rb.getString("FanWei")%>：1~64,Digit string</template>
										</el-input>
									</el-form-item>
									<el-form-item prop="left_auth" style="width:40%;min-width:400px;" label="Left Auth"  label-width="160px">
										<el-select v-model='addIPSecDialogForm.left_auth'>
											<el-option label='psk' value='psk'></el-option>
											<el-option label='pubkey' value='pubkey'></el-option>
											<el-option label='eap-aka' value='eap-aka'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="right_auth" style="width:40%;min-width:400px;" label="Right Auth"  label-width="160px">
										<el-select v-model='addIPSecDialogForm.right_auth'>
											<el-option label='psk' value='psk'></el-option>
											<el-option label='pubkey' value='pubkey'></el-option>
											<el-option label='eap-aka' value='eap-aka'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='gateway' style="width:40%;min-width:400px;" label='<%=rb.getString("WangGuan")%>' label-width="160px" class='validate-item'>
										<el-input v-model.trim='addIPSecDialogForm.gateway' maxlength="64">
											<template slot="append"><%=rb.getString("FanWei")%>：1~64,Digit string</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='right_subnet' style="width:40%;min-width:400px;" label="Right Subnet" label-width="160px" class='validate-item'>
										<el-input v-model.trim='addIPSecDialogForm.right_subnet' maxlength="128" >
											<template slot="append"><%=rb.getString("FanWei")%>：0~128,Digit string</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='right_id' style="width:40%;min-width:400px;" label="Right ID" label-width="160px" class='validate-item'>
										<el-input v-model.trim='addIPSecDialogForm.right_id' maxlength="64">
											<template slot="append"><%=rb.getString("FanWei")%>：0~64,Digit string</template>
										</el-input>
									</el-form-item>
									<el-form-item prop="secret_key" style="width:40%;min-width:400px;" label="SecretKey" label-width="110px" class='validate-item'>
										<el-input type="text" maxlength="64" v-model.trim='addIPSecDialogForm.secret_key'>
											<template slot="append"><%=rb.getString("FanWei")%>：0~64,Digit string</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='tunnel_left' style="width:40%;min-width:400px;" label="Left" label-width="160px" class='validate-item'>
										<el-input v-model.trim='addIPSecDialogForm.tunnel_left' maxlength="64"></el-input>
									</el-form-item>
									<el-form-item prop="right_secretkey" style="width:40%;min-width:400px;" label="Right Secret Key" label-width="110px" class='validate-item'>
										<el-input type="text" maxlength="64" v-model.trim='addIPSecDialogForm.right_secretkey'></el-input>
									</el-form-item>
								</div>
							</div>
						</el-collapse-item>
						<el-collapse-item name="Advance">
							<template slot='title'><p style="display:inline-block;margin-left:40px;"><span style="font-size:14px;font-weight:bold"><%=rb.getString("GaoJiSheZhi")%></span></p></template>
							<div class="rightContentCls">
								<div style="display:flex;margin-left:16px;flex-wrap: wrap">
									<el-form-item prop='left_id' style="width:40%;min-width:400px;" label="Left ID" label-width="160px" class='validate-item'>
										<el-input v-model.trim='addIPSecDialogForm.left_id' maxlength="64">
											<template slot="append"><%=rb.getString("FanWei")%>：0~64,Digit string</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='left_cert' style="width:40%;min-width:400px;" label="LeftCert" label-width="160px" class='validate-item'>
										<el-input v-model.trim='addIPSecDialogForm.left_cert' maxlength="64">
											<template slot="append"><%=rb.getString("FanWei")%>：0~64,Digit string</template>
										</el-input>
									</el-form-item>
									<el-form-item prop='left_source_ip' style="width:40%;min-width:400px;" label="LeftSourceIp" label-width="160px" class='validate-item'>
										<el-input v-model.trim='addIPSecDialogForm.left_source_ip' maxlength="64">
											<template slot="append"><%=rb.getString("FanWei")%>：0~64,Digit string</template>
										</el-input>
									</el-form-item>									
									<el-form-item prop='left_subnet' style="width:40%;min-width:400px;" label="Left Subnet" label-width="160px" class='validate-item'>
										<el-input v-model.trim='addIPSecDialogForm.left_subnet' maxlength="128">
											<template slot="append"><%=rb.getString("FanWei")%>：0~128,Digit string</template>
										</el-input>
									</el-form-item>
									<el-form-item prop="fragmentation" style="width:40%;min-width:400px;" label="Fragmentation" label-width="160px">
										<el-select v-model='addIPSecDialogForm.fragmentation'>
											<el-option label='Yes' value='yes'></el-option>
											<el-option label='Accept' value='accept'></el-option>
											<el-option label='Force' value='force'></el-option>
											<el-option label='No' value='no'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="ike_encryption" style="width:40%;min-width:400px;" label="IKE Encryption" label-width="160px">
										<el-select v-model='addIPSecDialogForm.ike_encryption'>
											<el-option label='aes128' value='aes128'></el-option>
											<el-option label='aes256' value='aes256'></el-option>
											<el-option label='3des' value='3des'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="ike_dh_group" style="width:40%;min-width:400px;" label="IKE DH Group" label-width="160px">
										<el-select v-model='addIPSecDialogForm.ike_dh_group'>
											<el-option label='modp768' value='modp768'></el-option>
											<el-option label='modp1024' value='modp1024'></el-option>
											<el-option label='modp1536' value='modp1536'></el-option>
											<el-option label='modp2048' value='modp2048'></el-option>
											<el-option label='modp4096' value='modp4096'></el-option>
											<el-option label='none' value='none'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="ike_authentication" style="width:40%;min-width:400px;" label="IKE Authentication" label-width="160px">
										<el-select v-model='addIPSecDialogForm.ike_authentication'>
											<el-option label='sha1' value='sha1'></el-option>
											<el-option label='sha1_160' value='sha1_160'></el-option>
											<el-option label='sha256_96' value='sha256_96'></el-option>
											<el-option label='sha256' value='sha256'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="esp_encryption" style="width:40%;min-width:400px;" label="ESP Encryption" label-width="160px">
										<el-select v-model='addIPSecDialogForm.esp_encryption'>
											<el-option label='aes128' value='aes128'></el-option>
											<el-option label='aes256' value='aes256'></el-option>
											<el-option label='3des' value='3des'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="esp_dh_group" style="width:40%;min-width:400px;" label="ESP DH Group" label-width="160px">
										<el-select v-model='addIPSecDialogForm.esp_dh_group'>
											<el-option label='modp768' value='modp768'></el-option>
											<el-option label='modp1024' value='modp1024'></el-option>
											<el-option label='modp1536' value='modp1536'></el-option>
											<el-option label='modp2048' value='modp2048'></el-option>
											<el-option label='modp4096' value='modp4096'></el-option>
											<el-option label='none' value='none'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="esp_authentication" style="width:40%;min-width:400px;" label="ESP Authentication" label-width="160px">
										<el-select v-model='addIPSecDialogForm.esp_authentication'>
											<el-option label='sha1' value='sha1'></el-option>
											<el-option label='sha1_160' value='sha1_160'></el-option>
											<el-option label='sha256_96' value='sha256_96'></el-option>
											<el-option label='sha256' value='sha256'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='key_lefe' style="width:40%;min-width:400px;" label="Key Lefe" label-width="160px" class='validate-item inputAndSelect'>
										<el-input v-model.trim='addIPSecDialogForm.key_lefe' style="width:152px" maxlength="64">
											<template v-if="addIPSecDialogForm.key_lefe_type == 's'" slot="append"><%=rb.getString("FanWei")%>：1~31536000,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.key_lefe_type == 'm'" slot="append"><%=rb.getString("FanWei")%>：1~525600,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.key_lefe_type == 'h'" slot="append"><%=rb.getString("FanWei")%>：1~8760,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.key_lefe_type == 'd'" slot="append"><%=rb.getString("FanWei")%>：1~365,<%=rb.getString("ZhengXing")%></template>
										</el-input>
										<el-select v-model='addIPSecDialogForm.key_lefe_type' style="position:absolute;left:144px;top:0px;">
											<el-option label='s' value='s'></el-option>
											<el-option label='m' value='m'></el-option>
											<el-option label='h' value='h'></el-option>
											<el-option label='d' value='d'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='ike_lefe_time' style="width:40%;min-width:400px;" label="IKELeftTime" label-width="160px" class='validate-item inputAndSelect'>
										<el-input v-model.trim='addIPSecDialogForm.ike_lefe_time' style="width:152px" maxlength="64">
											<template v-if="addIPSecDialogForm.ike_lefe_time_type == 's'" slot="append"><%=rb.getString("FanWei")%>：1~31536000,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.ike_lefe_time_type == 'm'" slot="append"><%=rb.getString("FanWei")%>：1~525600,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.ike_lefe_time_type == 'h'" slot="append"><%=rb.getString("FanWei")%>：1~8760,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.ike_lefe_time_type == 'd'" slot="append"><%=rb.getString("FanWei")%>：1~365,<%=rb.getString("ZhengXing")%></template>
										</el-input>
										<el-select v-model='addIPSecDialogForm.ike_lefe_time_type' style="position:absolute;left:144px;top:0px;">
											<el-option label='s' value='s'></el-option>
											<el-option label='m' value='m'></el-option>
											<el-option label='h' value='h'></el-option>
											<el-option label='d' value='d'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='rekey_margin' style="width:40%;min-width:400px;" label="RekeyMargin" label-width="160px" class='validate-item inputAndSelect'>
										<el-input v-model.trim='addIPSecDialogForm.rekey_margin' style="width:152px" maxlength="64">
											<template v-if="addIPSecDialogForm.rekey_margin_type == 's'" slot="append"><%=rb.getString("FanWei")%>：1~31536000,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.rekey_margin_type == 'm'" slot="append"><%=rb.getString("FanWei")%>：1~525600,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.rekey_margin_type == 'h'" slot="append"><%=rb.getString("FanWei")%>：1~8760,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.rekey_margin_type == 'd'" slot="append"><%=rb.getString("FanWei")%>：1~365,<%=rb.getString("ZhengXing")%></template>
										</el-input>
										<el-select v-model='addIPSecDialogForm.rekey_margin_type' style="position:absolute;left:144px;top:0px;">
											<el-option label='s' value='s'></el-option>
											<el-option label='m' value='m'></el-option>
											<el-option label='h' value='h'></el-option>
											<el-option label='d' value='d'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="dpdaction" style="width:40%;min-width:400px;" label="Dpdaction" label-width="160px">
										<el-select v-model='addIPSecDialogForm.dpdaction'>
											<el-option label='None' value='none'></el-option>
											<el-option label='Clear' value='clear'></el-option>
											<el-option label='Hold' value='hold'></el-option>
											<el-option label='Restart' value='restart'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop='dpddelay' style="width:40%;min-width:400px;" label="Dpddelay" label-width="160px" class='validate-item inputAndSelect'>
										<el-input v-model.trim='addIPSecDialogForm.dpddelay' style="width:152px" maxlength="64">
											<template v-if="addIPSecDialogForm.dpddelay_type == 's'" slot="append"><%=rb.getString("FanWei")%>：1~31536000,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.dpddelay_type == 'm'" slot="append"><%=rb.getString("FanWei")%>：1~525600,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.dpddelay_type == 'h'" slot="append"><%=rb.getString("FanWei")%>：1~8760,<%=rb.getString("ZhengXing")%></template>
											<template v-if="addIPSecDialogForm.dpddelay_type == 'd'" slot="append"><%=rb.getString("FanWei")%>：1~365,<%=rb.getString("ZhengXing")%></template>
										</el-input>
										<el-select v-model='addIPSecDialogForm.dpddelay_type' style="position:absolute;left:144px;top:0px;">
											<el-option label='s' value='s'></el-option>
											<el-option label='m' value='m'></el-option>
											<el-option label='h' value='h'></el-option>
											<el-option label='d' value='d'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="left_interface" style="width:40%;min-width:400px;" label="Left Interface" label-width="160px">
										<el-select v-model='addIPSecDialogForm.left_interface'>
											<el-option label='None' value=''></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="rekey" style="width:40%;min-width:400px;" label="REKEY" label-width="160px">
										<el-select v-model='addIPSecDialogForm.rekey'>
											<el-option label='No' value='No'></el-option>
											<el-option label='Yes' value='Yes'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="reauth" style="width:40%;min-width:400px;" label="REAUTH" label-width="160px">
										<el-select v-model='addIPSecDialogForm.reauth'>
											<el-option label='No' value='No'></el-option>
											<el-option label='Yes' value='Yes'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="forceencaps" style="width:40%;min-width:400px;" label="FORCEENCAPS" label-width="160px">
										<el-select v-model='addIPSecDialogForm.forceencaps'>
											<el-option label='No' value='No'></el-option>
											<el-option label='Yes' value='Yes'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item prop="mobike" style="width:40%;min-width:400px;" label="MOBIKE" label-width="160px">
										<el-select v-model='addIPSecDialogForm.mobike'>
											<el-option label='No' value='No'></el-option>
											<el-option label='Yes' value='Yes'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item v-if='false' prop='used_source_ip' style="width:40%;min-width:400px;" label="LeftSourceIp" label-width="160px" class='validate-item'>
										<el-input v-model.trim='addIPSecDialogForm.used_source_ip'></el-input>
									</el-form-item>
								</div>
							</div>
						</el-collapse-item>
					</el-collapse>
				</el-form> 
			</div>
			<div slot="footer" class="importFooter">
				<el-button type="primary" @click="addIPSecTunnelDialogSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="addIPSecTunnelDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
			</div>			
		</el-dialog><!--batch import-->
		<div class='rightBox' style='position: relative;' v-show='commonImportFileShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
				<span class='AddTitle commonText14'><%=rb.getString("DaoRu")%></span>
				<span class='closeIconBox' @click='gnbParamsImportCancel'><i class='el-icon el-icon-close'></i></span>
			</div>
			<div class='rightContent contentHeight' style='padding: 0 20px;'>
				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'><%=rb.getString("GNBDaoRuTiShi")%></span></div>
				<div class='originalVersionBox'>
					<el-form ref="gnbImportRuleForm" :model="gnbImportRuleForm" :rules="gnbImportRules" label-position="top" v-loading="commonImportFileLoading">
						<el-form-item label='<%=rb.getString("DaoRuLeiXing")%>' style="margin-bottom: 20px;">
							<el-radio-group v-model="gnbImportRuleForm.operType">
			                    <el-radio label="0" border><%=rb.getString("ZhuiJia")%></el-radio>
			                    <el-radio label="1" border><%=rb.getString("TiHuan")%></el-radio>
			                </el-radio-group>
						</el-form-item>
						<el-form-item label='<%=rb.getString("WenJian")%>' prop="fileName">
							<el-upload ref="upload" :before-upload='gnbBeforeUpload' :on-success='gnbCheckFile' :on-change="gnbFileChange" :show-file-list=false 	                  		
							    :action="gnbImportRuleForm.uploadFileUrl" :data="fileParams" name="uploadFile" accept=".xls,.xlsx" :auto-upload="false">
								<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
									<a slot="append" class="el-icon el-icon-operation-import importBox" @click="gnbFileSelect"></a>
								</el-input>								
								<a slot="trigger" ref="file_up"></a>
							</el-upload>
						</el-form-item>
						<div>
							<span class='commonNotes12' style='padding-bottom: 10px;display: block;'><%=rb.getString("DaoRuWenJianTiShi")%></span>
						    <span style="cursor:pointer;" @click="gnbExportTemplate">
							   <span class='el-icon el-icon-common-download'></span>
							   <span class='commonGeneral12' style='text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
						    </span>
					   </div>
					</el-form>
				</div>
			</div>
			<div class='commonFlex commonBorderTop commonFormFotter'>
				<div style='padding: 10px 30px 0;'>
					<el-button type="primary" @click="gnbParamsImportSubmit" :disabled='commonImportFileDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button @click="gnbParamsImportCancel"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>
		</div><!--批量导入失败窗口-->
		<el-dialog title='<%=rb.getString("XinXi")%>' width="400px" :visible="downloadVisible" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeDownloadDialog">
			<div>
				<p style="color:#797979;font-size:14px;"><%=rb.getString("BuFenShuJuCuoWuQingChongXingBianJi")%></p>
				<el-button style="margin-top:20px;margin-left:250px;" @click="downloadErrorFile" type="primary"><%=rb.getString("XiaZai")%></el-button>
			</div>
		</el-dialog><!--batch import: 修改 第一层弹窗-->
		<el-dialog class="gnbIpsecConfigAddDialog" title='<%=rb.getString("XiuGai")%>' top="20vh" width="60%" 
			:visible.sync="paramsImportEditDialogShow" @close="paramsImportEditDialogClose" :close-on-click-modal="false" append-to-body>
			<div class="gnbConfigAddMainBoxCls">
				<el-form ref="gnbImportAddOrEditForm" :model="gnbImportAddOrEditForm" :rules="gnbImportAddOrEditRules" label-position="top" label-width="125">
					<el-collapse v-model="gnbImportActiveName">
						<el-collapse-item name="gnb">
							<template slot='title'><p style="display:inline-block;margin-left: 40px;"><span class='commonText14'>GNB</span></p></template>
							<div class="rightContentCls">
								<div style='display:flex;margin-left:16px;flex-wrap: wrap'>
									<el-form-item label='<%=rb.getString("GNBMingCheng")%>' prop="gnbName" class='inputCommon validate-item commonParamImportItem'>
										<el-input v-model.trim="gnbImportAddOrEditForm.gnbName" maxlength="150">
											<template slot="append"><%=rb.getString("GNBChangDuTiShi")%>1~150 Digit</template>
										</el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("GNBBiaoShiChangDu")%>' prop="gnbLength" class='inputCommon validate-item commonParamImportItem'>
										<el-input v-model.trim="gnbImportAddOrEditForm.gnbLength">
											<template slot="append"><%=rb.getString("FanWei")%>：22~32,<%=rb.getString("ZhengXing")%></template>
										</el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("GNBBiaoShi")%>' prop="gnbId" class='inputCommon validate-item' style="width: 100%;">
										<el-input v-model.trim="gnbImportAddOrEditForm.gnbId">
											<template slot="append"><%=rb.getString("FanWei")%>：0~4294967295,<%=rb.getString("ZhengXing")%></template>
										</el-input>
										<span class='commonNotes12' style="display: block"><i class="el-icon el-icon-circle-info"></i><%=rb.getString("GNBCellZiDuanTiShi")%></span>
									</el-form-item>
								</div>
							</div>
						</el-collapse-item>
						<el-collapse-item name="cell">
							<template slot='title'><p style="display:inline-block;margin-left: 40px;"><span class='commonText14'>CELL</span></p></template>
							<div class="rightContentCls">
								<div style='display:flex;margin-left:16px;flex-wrap: wrap'>
									<el-form-item label="PCI" prop="pci" class='inputCommon validate-item' style="width: 100%;">
										<el-input v-model.trim="gnbImportAddOrEditForm.pci">
											<template slot="append"><%=rb.getString("FanWei")%>：0~1007,<%=rb.getString("ZhengXing")%></template>
										</el-input>
										<span class='commonNotes12' style="display: block"><i class="el-icon el-icon-circle-info"></i><%=rb.getString("GNBCELLPCIZiDuanTiShi")%></span>
									</el-form-item>
									<div class="commonFlex" style="height: 24px; line-height: 24px; padding-bottom: 20px;">
										<div @click="cellMoreItemShow = !cellMoreItemShow" style="cursor:pointer;">
											<span class="commonText14"><%=rb.getString("GNBGengDuo")%></span>
											<span class="el-icon el-icon-circle-down" :class="cellMoreItemShow ? 'el-icon-circle-up' : 'el-icon-circle-down'" style="padding: 0 10px;"></span>
				 						</div>
										<span @click="clearParamsValue('importForm')" class='commonParamBtn'><%=rb.getString("QingKong")%></span>
										<span @click="defaultParamsValue('importForm')" class='commonParamBtn' style='margin: 0 10px;'><%=rb.getString("APNMoRen")%></span>
										<span class="commonNotes12"><%=rb.getString("GNBZiDuanShuRuTiShi")%></span>
										<span class='tipLine'></span>
									</div>
									<div style='width: 100%; padding-top: 10px;' class='' v-show="cellMoreItemShow">
										<el-form-item label='<%=rb.getString("SSBPinDianHao")%>' prop="ssbFrequency" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.ssbFrequency">
												<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("GNBPinDaiZhiShi")%>' prop="freqBandIndicator" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.freqBandIndicator">
												<template slot="append"><%=rb.getString("FanWei")%>：1~1024,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("XiaXingNRARFCN")%>' prop="nrarfcndl" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.nrarfcndl">
												<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("ShangXingNRARFCN")%>' prop="nrarfcnul" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm .nrarfcnul">
												<template slot="append"><%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("GNBXiaXingDaiKuan")%>' prop="dlbandwidth" class='inputCommon validate-item commonParamImportItem'>
											<el-select v-model="gnbImportAddOrEditForm.dlbandwidth">
												<el-option label="5MHz" value="5"></el-option>
												<el-option label="10MHz" value="10"></el-option>
												<el-option label="15MHz" value="15"></el-option>
												<el-option label="20MHz" value="20"></el-option>
												<el-option label="25MHz" value="25"></el-option>
												<el-option label="30MHz" value="30"></el-option>
												<el-option label="40MHz" value="40"></el-option>
												<el-option label="50MHz" value="50"></el-option>
												<el-option label="60MHz" value="60"></el-option>
												<el-option label="80MHz" value="80"></el-option>
												<el-option label="90MHz" value="90"></el-option>
												<el-option label="100MHz" value="100"></el-option>
												<el-option label="200MHz" value="200"></el-option>
												<el-option label="400MHz" value="400"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label='<%=rb.getString("GNBShangXingDaiKuan")%>' prop="ulbandwidth" class='inputCommon validate-item commonParamImportItem'>
											<el-select v-model="gnbImportAddOrEditForm.ulbandwidth">
												<el-option label="5MHz" value="5"></el-option>
												<el-option label="10MHz" value="10"></el-option>
												<el-option label="15MHz" value="15"></el-option>
												<el-option label="20MHz" value="20"></el-option>
												<el-option label="25MHz" value="25"></el-option>
												<el-option label="30MHz" value="30"></el-option>
												<el-option label="40MHz" value="40"></el-option>
												<el-option label="50MHz" value="50"></el-option>
												<el-option label="60MHz" value="60"></el-option>
												<el-option label="80MHz" value="80"></el-option>
												<el-option label="90MHz" value="90"></el-option>
												<el-option label="100MHz" value="100"></el-option>
												<el-option label="200MHz" value="200"></el-option>
												<el-option label="400MHz" value="400"></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label='<%=rb.getString("JiZhanZhiShi")%>' prop="duplex_mode" class='inputCommon validate-item commonParamImportItem'>
											<el-select v-model='gnbImportAddOrEditForm.duplex_mode'>
												<el-option label='TDD' value='TDDMode'></el-option>
												<el-option label="FDD" value='FDDMode'></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label='<%=rb.getString("GNBXiaXingZiZaiBoJianGe")%>' prop="dlSubcarrierSpacing" class='inputCommon validate-item commonParamImportItem'>
											<el-select v-model='gnbImportAddOrEditForm.dlSubcarrierSpacing'>
												<el-option label="30kHz" value='1' v-if="gnbImportAddOrEditForm.duplex_mode == 'TDDMode'"></el-option>
												<el-option label="15kHz" value='0' v-else></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label='<%=rb.getString("GNBShangXingXingZiZaiBoJianGe")%>' prop="ulSubcarrierSpacing" class='inputCommon validate-item commonParamImportItem'>
											<el-select v-model='gnbImportAddOrEditForm.ulSubcarrierSpacing'>
												<el-option label="30kHz" value='1' v-if="gnbImportAddOrEditForm.duplex_mode == 'TDDMode'"></el-option>
												<el-option label="15kHz" value='0' v-else></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label='<%=rb.getString("GNBFaSheTongDaoShu")%>' prop="ulAntNum" class='inputCommon validate-item commonParamImportItem'>
											<el-select v-model='gnbImportAddOrEditForm.ulAntNum'>
												<el-option label="1" value='1'></el-option>
												<el-option label='2' value='2'></el-option>
												<el-option label="4" value='4'></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label='<%=rb.getString("GNBJieShouTongDaoShu")%>' prop="dlAntNum" class='inputCommon validate-item commonParamImportItem'>
											<el-select v-model='gnbImportAddOrEditForm.dlAntNum'>
												<el-option label="1" value='1'></el-option>
												<el-option label='2' value='2'></el-option>
												<el-option label="4" value='4'></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label='<%=rb.getString("MoShi1ChunShuZhouQi")%>' prop="dlulTransmissionPeriodicity1" class='inputCommon validate-item commonParamImportItem'>
											<el-select v-model='gnbImportAddOrEditForm.dlulTransmissionPeriodicity1'>
												<el-option label="0" value='0'></el-option>
												<el-option label="1" value='1'></el-option>
												<el-option label='2' value='2'></el-option>
												<el-option label="3" value='3'></el-option>
												<el-option label="4" value='4'></el-option>
												<el-option label='5' value='5'></el-option>
												<el-option label="6" value='6'></el-option>
												<el-option label="7" value='7'></el-option>
												<el-option label="8" value='8'></el-option>
												<el-option label="9" value='9'></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label='<%=rb.getString("MoShi1LianXuXiaXingXiShu")%>' prop="nrofDownlinkSlots1" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.nrofDownlinkSlots1">
												<template slot="append"><%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("MoShi1LianXuShangXingXiShu")%>' prop="nrofUplinkSlots1" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.nrofUplinkSlots1">
												<template slot="append"><%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("MoShi1LianXuXiaXingFuHaoShu")%>' prop="nrofDownlinkSymbols1" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.nrofDownlinkSymbols1">
												<template slot="append"><%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("MoShi1LianXuShangXingFuHaoShu")%>' prop="nrofUplinkSymbols1" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.nrofUplinkSymbols1">
												<template slot="append"><%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("MoShi2ChunShuZhouQi")%>' prop="dlulTransmissionPeriodicity2" class='inputCommon validate-item commonParamImportItem'>
											<el-select v-model='gnbImportAddOrEditForm.dlulTransmissionPeriodicity2'>
												<el-option label='<%=rb.getString("Guan")%>' value='4095'></el-option>
												<el-option label="0" value='0'></el-option>
												<el-option label="1" value='1'></el-option>
												<el-option label='2' value='2'></el-option>
												<el-option label="3" value='3'></el-option>
												<el-option label="4" value='4'></el-option>
												<el-option label='5' value='5'></el-option>
												<el-option label="6" value='6'></el-option>
												<el-option label="7" value='7'></el-option>
												<el-option label="8" value='8'></el-option>
												<el-option label="9" value='9'></el-option>
											</el-select>
										</el-form-item>
										<el-form-item v-if="gnbImportAddOrEditForm.dlulTransmissionPeriodicity2 != '4095'" label='<%=rb.getString("MoShi2LianXuXiaXingXiShu")%>' prop="nrofDownlinkSlots2" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.nrofDownlinkSlots2">
												<template slot="append"><%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item v-if="gnbImportAddOrEditForm.dlulTransmissionPeriodicity2 != '4095'" label='<%=rb.getString("MoShi2LianXuShangXingXiShu")%>' prop="nrofUplinkSlots2" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.nrofUplinkSlots2">
												<template slot="append"><%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item v-if="gnbImportAddOrEditForm.dlulTransmissionPeriodicity2 != '4095'" label='<%=rb.getString("MoShi2LianXuXiaXingFuHaoShu")%>' prop="nrofDownlinkSymbols2" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.nrofDownlinkSymbols2">
												<template slot="append"><%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<el-form-item v-if="gnbImportAddOrEditForm.dlulTransmissionPeriodicity2 != '4095'" label='<%=rb.getString("MoShi2LianXuShangXingFuHaoShu")%>' prop="nrofUplinkSymbols2" class='inputCommon validate-item commonParamImportItem'>
											<el-input v-model.trim="gnbImportAddOrEditForm.nrofUplinkSymbols2">
												<template slot="append"><%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>										
										<el-form-item label='<%=rb.getString("GNBGenXuLieSuoYin")%>' prop="prachRootSequenceIndex" class='inputCommon validate-item commonParamImportItem'>
											<el-select v-model='gnbImportAddOrEditForm.prachRootSequenceIndex'>
												<el-option label="0" value='0'></el-option>
												<el-option label="1" value='1'></el-option>
											</el-select>
										</el-form-item>
										<el-form-item label='<%=rb.getString("GNBGenXuLieZhi")%>' prop="prachRootSequenceValue" class='inputCommon validate-item' style='margin-bottom: 10px;'>
											<el-input v-model.trim="gnbImportAddOrEditForm.prachRootSequenceValue">
												<template slot="append" v-if='gnbImportAddOrEditForm.prachRootSequenceIndex =="0"'><%=rb.getString("FanWei")%>：0~837,<%=rb.getString("ZhengXing")%></template>
												<template slot="append" v-if='gnbImportAddOrEditForm.prachRootSequenceIndex =="1"'><%=rb.getString("FanWei")%>：0~137,<%=rb.getString("ZhengXing")%></template>
											</el-input>
										</el-form-item>
										<span class='commonNotes12' style="display: block"><i class="el-icon el-icon-circle-info"></i><%=rb.getString("GNBZiDuanShuRuTiShi2")%></span>
									</div>
								</div>
							</div>
						</el-collapse-item>
						<el-collapse-item name="plmn">
							<template slot='title'><p style="display:inline-block;margin-left: 40px;"><span class='commonText14'>PLMN</span></p></template>
							<div class="rightContentCls">
								<div style='display:flex;margin-left:16px;flex-wrap: wrap'>
									<div style='width: 100%;' class='paramPoolWarp'>
										<div class="contentTableTitle" style="padding-bottom: 5px;">
											<div >
												<span class="commonSize14"><%=rb.getString("PLMNBiaoShiXinXiSheZhi")%></span>
												<span class="commonNotes12" style="margin-left: 5px;"><%=rb.getString("BuChaoGuo6Ge")%></span>
											</div><!--row,operType,模块, 父级数据(plmn nr)-->
											<div class='licenseImportBox' style="margin: 0" v-show='gnbImportAddOrEditForm.plmn.length < 6' @click="importParamClick('','add','importPlmn','')">
												<i class='el-icon el-icon-circle-add importIcon'></i>
											</div>
										</div>
										<div style="height: fit-content; padding-bottom: 18px;" >
											<el-table ref="importPlmnTable" :rownumber="true" id="importPlmnTable" :data="gnbImportAddOrEditForm.plmn" :pagination="false" :row-key="'index'" default-expand-all style="border:1px solid #E9E9E9; min-height: 200px;">
												<el-table-column type="expand">
													<template slot-scope="props">
														<div style='padding: 10px 0px;'>
															<div style="padding-bottom: 5px; display: flex;width: 100%;">
																<span class="commonSize14"><%=rb.getString("GNBPLMNSheZhi")%></span>
																<span class="commonNotes12" style="margin-left: 5px;"><%=rb.getString("BuChaoGuo6Ge")%></span>
																<!--row,operType,模块, 父级数据(plmn nr)-->
																<span class="el-icon el-icon-circle-add" v-show='props.row.plmnConfigList.length < 6' @click="importParamClick('','add','importPlmnConfig', props.row)" style="margin-left: 10px;"></span>
															</div>
															<div v-if='props.row.plmnConfigList'>
																<el-table ref="importPlmnConfigTable" :rownumber="true" id="importPlmnConfigTable" :data="props.row.plmnConfigList" :pagination="false" default-expand-all style="border:1px solid #E9E9E9;">
																	<el-table-column type="expand">
																		<template slot-scope="props">
																			<div style='padding: 10px 0px;'>
																				<div style="padding-bottom: 5px; display: flex;width: 100%;">
																					<span class="commonSize14"><%=rb.getString("GNBQiePianLieBiao")%></span>
																					<span class="commonNotes12" style="margin-left: 5px;"><%=rb.getString("BuChaoGuo6Ge")%></span>
																					<span class="el-icon el-icon-circle-add" v-show='props.row.sliceList.length < 6' @click="importParamClick('','add','importSliceConfig', props.row)" style="margin-left: 10px;"></span>
																				</div>
																				<div v-if='props.row.sliceList'>
																					<el-table ref="importSliceListTable" :rownumber="true" id="importSliceListTable" :data="props.row.sliceList" :pagination="false" style="border:1px solid #E9E9E9;">
																						<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
																							<template slot-scope="scope">
																								<span class="el-icon el-icon-operation-edit" @click="importParamClick(scope.row,'edit','importSliceConfig',props.row)" style="margin-right:15px;"></span>
																								<span class="el-icon el-icon-operation-delete" @click="importParamDelClick(scope.row, 'importSliceConfig', props.row)" ></span>
																							</template>
																						</el-table-column>
																						<el-table-column label='ID' min-width="120" prop="index" show-overflow-tooltip></el-table-column>
																						<el-table-column label='SD' min-width="120" prop="sd" show-overflow-tooltip>
																							<template slot-scope="scope">
																								<div v-if="scope.row.sd === '1'"><%=rb.getString("GNBFeiKong")%></div>
																								<div v-else><%=rb.getString("GNBKong")%></div>
																							</template>
																						</el-table-column>
																						<el-table-column label='SNSSAI' min-width="120" prop="sd_value" show-overflow-tooltip></el-table-column>
																					</el-table>
																				</div>
																			</div>
																		</template>
																	</el-table-column>
																	<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
																		<template slot-scope="scope">
																			<span class="el-icon el-icon-operation-edit" @click="importParamClick(scope.row,'edit','importPlmnConfig', props.row)" style="margin-right:15px;"></span>
																			<span class="el-icon el-icon-operation-delete" @click="importParamDelClick(scope.row, 'importPlmnConfig', props.row)" ></span>
																		</template>
																	</el-table-column>
																	<el-table-column label='ID' prop="index" show-overflow-tooltip></el-table-column>
																	<el-table-column label='<%=rb.getString("GNBPLMNBiaoShi")%>' prop="plmnId" show-overflow-tooltip></el-table-column>
																	<el-table-column label='<%=rb.getString("GNBZhuPLMN")%>' prop="primary" show-overflow-tooltip></el-table-column>
																</el-table>
															</div>
														</div>
													</template>
												</el-table-column>
												<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
													<template slot-scope="scope">
														<span class="el-icon el-icon-operation-edit" @click="importParamClick(scope.row,'edit','importPlmn', '')" style="margin-right:15px;"></span>
														<span class="el-icon el-icon-operation-delete" @click="importParamDelClick(scope.row, 'importPlmn', '')" ></span>
													</template>
												</el-table-column>
												<el-table-column label='ID' min-width="120" prop="index" show-overflow-tooltip></el-table-column>
												<el-table-column label='<%=rb.getString("GNBNCI")%>' min-width="120" prop="NCI" show-overflow-tooltip></el-table-column>
												<el-table-column label='<%=rb.getString("GNBGenZongQuYuMa")%>' min-width="120" prop="tac" show-overflow-tooltip></el-table-column>
												<el-table-column label='<%=rb.getString("GNBWuXianJieRuWangQuYuMa")%>' min-width="120" prop="ranac" show-overflow-tooltip></el-table-column>
											</el-table>
											<el-form-item prop='plmn' style="margin: 0 0 20px 0;" label-width="0" class='plmnLengthValidate'>
												<el-input v-model='gnbImportAddOrEditForm.plmn' v-show="false"></el-input>
											</el-form-item>
										</div>
									</div>
								</div>
							</div>
						</el-collapse-item><!--Advance-->
						<el-collapse-item name="advance">
							<template slot='title'><p style="display:inline-block;margin-left: 40px;"><span class='commonText14'><%=rb.getString("Advance")%></span></p></template>
							<div class="rightContentCls">
								<div class='commonFlex commonContent' style="justify-content: flex-start;">
									<span class='commonGeneralBold12 ' style='padding: 6px 0;'><%=rb.getString("GNBCanShuPeiZhi")%></span>	
									<span class='commonTitle12' style='padding: 6px;'><%=rb.getString("GaoJingXuanZeCanShuPeiZhi")%></span>
								</div>
								<div style='margin-left:16px;'>
									<div>
										<div class='commonFlex commonContent' style="justify-content: flex-start;">
											<span class='tieleTipCircle'></span>
											<span class='commonText14' style='padding: 6px 0;'>AMF</span>		
										</div> 
										<div style='padding: 5px 0px 20px 18px;' class='paramPoolWarp'>
											<div class="contentTableTitle" style="padding-bottom: 5px;">
												<div class="commonSize14"><%=rb.getString("AMFIPLieBiao")%></div>
												<div class='licenseImportBox' @click="importParamClick('', 'add', 'importAMF', '')" style="margin: 0">
													<i class='el-icon el-icon-plus importIcon'></i>
												</div>
											</div>
											<el-ctable ref="importAmfTable" :rownumber="true" id="importAmfTable" :data="gnbImportAddOrEditForm.amf" height="200px" :pagination="false" style="border:1px solid #E9E9E9;">
												<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
													<template slot-scope="scope">
														<span class="el-icon el-icon-operation-delete" @click="importParamDelClick(scope.row, 'importAMF', '')"></span>
													</template>
												</el-table-column>
												<el-table-column label='ID' min-width="40" prop="index" show-overflow-tooltip></el-table-column>
												<el-table-column label='AMF IP' min-width="120" prop="amfIp" show-overflow-tooltip></el-table-column>
												<el-table-column label='<%=rb.getString("GNBPLMNBiaoShi")%>' min-width="120" prop="plmnId" show-overflow-tooltip></el-table-column>
												<el-table-column label='<%=rb.getString("APNMoRen")%>' min-width="120" prop="default" show-overflow-tooltip></el-table-column>
											</el-ctable>
											<el-form-item prop='amf' style="display:none;" label="" label-width="0px">
												<el-input v-model='gnbImportAddOrEditForm.amf'></el-input>
											</el-form-item>	
										</div>
									</div>
									<div><!--Network interfaceSetting-->
										<div class='commonFlex commonContent' style="justify-content: flex-start;">
											<span class='tieleTipCircle'></span>
											<span class='commonText14' style='padding: 6px 0;'><%=rb.getString("JieKouSheZhi")%></span>		
										</div>
										<div style='padding: 5px 0px 0px 18px;' class='paramPoolWarp'>
											<div class="contentTableTitle" style="padding-bottom: 5px;">
												<div class="commonSize14"><%=rb.getString("WANLANlieBiao")%></div>
												<div class='licenseImportBox' @click="importParamClick('', 'add', 'importWan', '')" style="margin: 0"><i class='el-icon el-icon-plus importIcon'></i></div>
											</div>
											<div class="cellTableBoxCls" style="padding-bottom:20px;">
												<el-ctable ref="importWanListTable" :rownumber="true" id="importWanListTable" :data="gnbImportAddOrEditForm.wan" height="200px" :pagination="false" style="border:1px solid #E9E9E9;">
													<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
														<template slot-scope="scope">
															<span class="el-icon el-icon-operation-edit" @click="importParamClick(scope.row, 'edit', 'importWan', '')" style="margin-right:15px;"></span>
															<span class="el-icon el-icon-operation-delete" @click="importParamDelClick(scope.row, 'importWan', '')" ></span>
														</template>
													</el-table-column>
													<el-table-column label='<%=rb.getString("IPLeiXing")%>' prop="addressType" min-width="100" show-overflow-tooltip>
														<template slot-scope="scope">
															<div v-if="scope.row.addressType == 'DHCP'">DHCP</div>
															<div v-if="scope.row.addressType == 'Static'">Static</div>
															<div v-if="scope.row.addressType == 'DHCPv6'">IPv6 DHCP</div>
															<div v-if="scope.row.addressType == 'Staticv6'">IPv6 Static</div>
														</template>
													</el-table-column>
													<el-table-column label='<%=rb.getString("IPDiZhi")%>' prop="ipAddress" min-width="120" show-overflow-tooltip></el-table-column>
													<el-table-column label='<%=rb.getString("QianZhuiChangDuZiWangYanMa")%>' prop="subnetMask" min-width="200" show-overflow-tooltip>
														<template slot-scope="scope">
															<span v-if="scope.row.addressType && scope.row.addressType == 'Static'">{{scope.row.subnetMask}}</span>
															<span v-else-if="scope.row.addressType && scope.row.addressType == 'Staticv6'">{{scope.row.prefixLength}}</span>
															<span v-else></span>
														</template>
													</el-table-column>
													<el-table-column label='<%=rb.getString("WangGuan")%>' prop="gateway" min-width="120" show-overflow-tooltip></el-table-column>
													<el-table-column label='<%=rb.getString("ChengZaiLeiXing")%>' prop="bearType" min-width="100" show-overflow-tooltip></el-table-column>
													<el-table-column label='<%=rb.getString("VLANMingCheng")%>' min-width="100" prop="vlanName" show-overflow-tooltip></el-table-column>
													<el-table-column label='VLAN ID' min-width="80" prop="vlanId" show-overflow-tooltip></el-table-column>
												</el-ctable>
												<el-form-item prop='wan' style="display:none;" label="" label-width="0px">
													<el-input v-model='gnbImportAddOrEditForm.wan'></el-input>
												</el-form-item>
											</div>
										</div>
										<div style='padding: 5px 0px 0px 18px;' class='paramPoolWarp'>
											<div class="contentTableTitle" style="padding-bottom: 5px;">
												<div class="commonSize14"><%=rb.getString("LANLieBiao")%></div>
												<div class='licenseImportBox' @click="importParamClick('', 'add', 'importLan', '')" style="margin: 0"><i class='el-icon el-icon-plus importIcon'></i></div>
											</div>
											<div class="cellTableBoxCls" style="padding-bottom:20px;">
												<el-ctable ref="importLanListTable" :rownumber="true" id="importLanListTable" :data="gnbImportAddOrEditForm.lan" height="200px" :pagination="false" style="border:1px solid #E9E9E9;">
													<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
														<template slot-scope="scope">
															<span class="el-icon el-icon-operation-edit" @click="importParamClick(scope.row, 'edit', 'importLan', '')" style="margin-right:15px;"></span>
															<span class="el-icon el-icon-operation-delete" @click="importParamDelClick(scope.row, 'importLan', '')" ></span>
														</template>
													</el-table-column>
													<el-table-column label='ID' min-width="120" prop="index" show-overflow-tooltip></el-table-column>
													<el-table-column label='<%=rb.getString("IPDiZhi")%>' min-width="120" prop="ipAddress" show-overflow-tooltip></el-table-column>
													<el-table-column label='<%=rb.getString("GNBZiWangYanMa")%>' min-width="120" prop="subnetMask" show-overflow-tooltip></el-table-column>
												</el-ctable>
												<el-form-item prop='lan' style="display:none;" label="" label-width="0px">
													<el-input v-model='gnbImportAddOrEditForm.lan'></el-input>
												</el-form-item>
											</div>
										</div>
									</div>
									<div><!--Management Server-->
										<div class='commonFlex commonContent' style="justify-content: flex-start;">
											<span class='tieleTipCircle'></span>
											<span class='commonText14' style='padding: 6px 0;'><%=rb.getString("GuanLiFuWu")%></span>		
										</div> 
										<div class="rightContentCls">
											<div style='display:flex;margin-left:16px;flex-wrap: wrap' class='paramPoolWarpThree'>
												<el-form-item prop="periodicInformEnable" class='inputCommon validate-item' style='min-width: 300px;'>
													<div slot="label">
														<span class="commonNotes12" style="margin-left: 5px;color: #f56c6c;">*</span>
														<span class="commonSize14"><%=rb.getString("DingQiTongZhiKaiGuan")%></span>
													</div>
													<el-switch onclick="event.stopPropagation()" v-model="gnbImportAddOrEditForm.periodicInformEnable" active-value="1" inactive-value="0">
													</el-switch>
												</el-form-item>
												<el-form-item label='<%=rb.getString("DingQiTongZhiShiJian")%>' prop="periodicInformTime" class='inputCommon' style='min-width: 300px;'>
													<el-date-picker style='width: 200px;' v-model="gnbImportAddOrEditForm.periodicInformTime" type="datetime" value-format="yyyy-MM-dd HH:mm:ss">
													</el-date-picker>
												</el-form-item>
												<el-form-item label='<%=rb.getString("DingQiTongZhiJianGe")%>' prop="periodicInformInterval" class='inputCommon validate-item' style='min-width: 300px;'>
													<el-input v-model.trim="gnbImportAddOrEditForm.periodicInformInterval">
														<template slot="append"><%=rb.getString("FanWei")%>：1~Infinite,<%=rb.getString("ZhengXing")%></template>
													</el-input>
												</el-form-item>
											</div>
										</div>
									</div>
									<div>
										<div class='commonFlex commonContent' style="justify-content: flex-start;">
											<span class='tieleTipCircle'></span>
											<span class='commonText14' style='padding: 6px 0;'>NTP</span>		
										</div>
										<div class="rightContentCls">
											<div style='display:flex;margin-left:16px;flex-wrap: wrap' class='paramPoolWarpThree'>
												<el-form-item prop="ntpEnable" style='min-width: 300px;'>
													<div slot="label">
														<span class="commonNotes12" style="margin-left: 5px;color: #f56c6c;">*</span>
														<span class="commonSize14"><%=rb.getString("NTPKaiGuan")%></span>
													</div>
													<el-switch onclick="event.stopPropagation()" v-model="gnbImportAddOrEditForm.ntpEnable" active-value="1" inactive-value="0"></el-switch>
												</el-form-item>
												<el-form-item label='<%=rb.getString("GNBShiQu")%>' prop="localTimeZone" class='inputCommon' style='min-width: 300px;'>
													<el-select v-model='gnbImportAddOrEditForm.localTimeZone'>
														<el-option v-for="item in timeZoneList" :key="item.value" :label="item.label" :value="item.value"></el-option>
													</el-select>
												</el-form-item>
												<el-form-item label='<%=rb.getString("NTPFuWuQi")%> 1' prop="ntpServer1" class='inputCommon' style='min-width: 300px;'>
													<el-input v-model.trim="gnbImportAddOrEditForm.ntpServer1"></el-input>
												</el-form-item>
												<el-form-item label='<%=rb.getString("NTPFuWuQi")%> 2' prop="ntpServer2" class='inputCommon' style='min-width: 300px;'>
													<el-input v-model.trim="gnbImportAddOrEditForm.ntpServer2"></el-input>
												</el-form-item>
												<el-form-item label='<%=rb.getString("NTPFuWuQi")%> 3' prop="ntpServer3" class='inputCommon' style='min-width: 300px;'>
													<el-input v-model.trim="gnbImportAddOrEditForm.ntpServer3"></el-input>
												</el-form-item>
												<el-form-item label='<%=rb.getString("NTPFuWuQi")%> 4' prop="ntpServer4" class='inputCommon' style='min-width: 300px;'>
													<el-input v-model.trim="gnbImportAddOrEditForm.ntpServer4"></el-input>
												</el-form-item>
												<el-form-item label='<%=rb.getString("NTPFuWuQi")%> 5' prop="ntpServer5" class='inputCommon' style='min-width: 300px;'>
													<el-input v-model.trim="gnbImportAddOrEditForm.ntpServer5"></el-input>
												</el-form-item>
											</div> 
										</div>
									</div>
								</div>
							</div>
						</el-collapse-item>
					</el-collapse>
				</el-form>
			</div>
			<div slot="footer" class="importFooter">
				<el-button type="primary" @click="paramsImportEditDialogSubmit" :disabled='gnbParamImportSaveBtnDisabled'><%=rb.getString("QueDing")%></el-button>
				<el-button @click="paramsImportEditDialogClose"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</el-dialog><!--batch import:修改 弹窗中的按钮操作： 新建，修改 第二层 弹框-->
		<el-dialog class="gnbImportConfigAddDialog" :title='batchImportDialogTitle' top="25vh" width="50%" :visible.sync="paramsImportSaveDialogShow" 
			@close="paramsImportSaveDialogClose" :close-on-click-modal="false" append-to-body>
			<div class="gnbConfigAddMainBoxCls">
				<el-form ref="gnbImportSaveForm" :model="gnbImportSaveForm" :rules="gnbImportSaveFormRules" label-position="top" style='min-height: 160px;'>
					<div v-if="commonImportOperModel == 'importPlmn'">
						<span class='commonText14'><%=rb.getString("PLMNBiaoShiXinXiSheZhi")%></span>
						<div style='display:flex; padding-top: 20px;flex-wrap: wrap'>
							<el-form-item prop='NCI' class="titleAndNoteBox commonParamImportItem" >
								<div slot="label">
									<span class="commonSize14"><%=rb.getString("GNBNCI")%></span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：0~68719476735,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='gnbImportSaveForm.NCI'></el-input>
							</el-form-item>
							<el-form-item prop='tac' class="titleAndNoteBox commonParamImportItem" >
								<div slot="label">
									<span class="commonSize14"><%=rb.getString("GNBGenZongQuYuMa")%></span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：0~16777215,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='gnbImportSaveForm.tac'></el-input>
							</el-form-item>
							<el-form-item prop='ranac' class="titleAndNoteBox commonParamImportItem" >
								<div slot="label">
									<span class="commonSize14"><%=rb.getString("GNBWuXianJieRuWangQuYuMa")%></span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：0~255,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='gnbImportSaveForm.ranac'></el-input>
							</el-form-item>
						</div>
					</div>
					<div v-if="commonImportOperModel == 'importPlmnConfig'">
						<span class='commonText14'><%=rb.getString("GNBPLMNSheZhi")%></span>
						<div style='display:flex; padding-top: 20px;flex-wrap: wrap'>
							<el-form-item prop='plmnId' class="titleAndNoteBox commonParamImportItem" >
								<div slot="label">
									<span class="commonSize14"><%=rb.getString("GNBPLMNBiaoShi")%></span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：5~6 Digit,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='gnbImportSaveForm.plmnId'></el-input>
							</el-form-item>
							<el-form-item label='<%=rb.getString("GNBZhuPLMN")%>' prop='primary' class="titleAndNoteBox commonParamImportItem">
								<el-select v-model='gnbImportSaveForm.primary'>
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
								</el-select>
							</el-form-item>
						</div>
					</div>
					<div v-if="commonImportOperModel == 'importSliceConfig'">
						<span class='commonText14'>Slice Setting</span>
						<div style='display:flex; padding-top: 20px;flex-wrap: wrap'>
							<el-form-item label="SD" prop='sd' class="titleAndNoteBox commonParamImportItem">
								<el-radio-group v-model="gnbImportSaveForm.sd">
									<el-radio label="0" border><%=rb.getString("GNBKong")%></el-radio>
									<el-radio label="1" border><%=rb.getString("GNBFeiKong")%></el-radio>
								</el-radio-group>
							</el-form-item>
							<el-form-item prop='sd_value' v-if='gnbImportSaveForm.sd == "1"'  class="titleAndNoteBox commonParamImportItem" >
								<div slot="label">
									<span class="commonSize14">SNSSAI</span>
									<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：0~16777215,<%=rb.getString("ZhengXing")%>)</span>
								</div>
								<el-input v-model.trim='gnbImportSaveForm.sd_value'></el-input>
							</el-form-item>
						</div>
					</div>
					<div v-if="commonImportOperModel == 'importAMF'" style='display:flex;margin-left:16px;flex-wrap: wrap'>
						<el-form-item prop='amfIp' class="titleAndNoteBox commonParamImportItem" >
							<div slot="label">
								<span class="commonSize14">AMF IP</span>
								<span class="commonNotes12" style="margin-left: 5px;">(Example：1.1.1.1)</span>
							</div>
							<el-input v-model.trim='gnbImportSaveForm.amfIp'></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("GNBPLMNBiaoShi")%>' prop='plmnId' class="titleAndNoteBox commonParamImportItem">
							<el-select v-model='gnbImportSaveForm.plmnId' >
								<el-option v-for="item in importPlmnIdList" :label='item' :value='item'></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label='<%=rb.getString("APNMoRen")%>' prop='default' class='commonParamImportItem'>
							<el-select v-model='gnbImportSaveForm.default'>
								<el-option label='0' value='0'></el-option>
								<el-option label='1' value='1'></el-option>
							</el-select>
						</el-form-item>
					</div>
					<div v-if="commonImportOperModel == 'importWan'" style='display:flex;margin-left:16px;flex-wrap: wrap'>
						<el-form-item label='<%=rb.getString("IPLeiXing")%>' prop='addressType' class="titleAndNoteBox commonParamImportItem" >
							<el-select v-model='gnbImportSaveForm.addressType'>
								<el-option label='DHCP' value='DHCP'></el-option>
								<el-option label='Static' value='Static'></el-option>
								<el-option label='IPv6 DHCP' value='DHCPv6'></el-option>
								<el-option label='IPv6 Static' value='Staticv6'></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label='<%=rb.getString("ChengZaiLeiXing")%>' prop='bearType' class="titleAndNoteBox commonParamImportItem" >
							<el-select v-model='gnbImportSaveForm.bearType'>
								<el-option label='NG' value='NG'></el-option>
								<el-option label='OAM' value='OAM'></el-option>
								<el-option label='NGU' value='NGU'></el-option>
								<el-option label='NG/OAM' value='NG/OAM'></el-option>
								<el-option label='NG/NGU' value='NG/NGU'></el-option>
								<el-option label='NGU/OAM' value='NGU/OAM'></el-option>
								<el-option label='NG/OAM/NGU' value='NG/OAM/NGU'></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="IP" v-if="['Static','Staticv6'].includes(gnbImportSaveForm.addressType)" prop='ipAddress' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gnbImportSaveForm.ipAddress'></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("GNBZiWangYanMa")%>' v-if="['Static'].includes(gnbImportSaveForm.addressType )" prop='subnetMask' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gnbImportSaveForm.subnetMask'></el-input>
						</el-form-item>
						<el-form-item v-if="['Staticv6'].includes(gnbImportSaveForm.addressType )" prop='prefixLength' class="titleAndNoteBox" class="titleAndNoteBox commonParamImportItem">
							<div slot="label">
								<span class="commonSize14"><%=rb.getString("QianZhuiChangDu")%></span>
								<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：0~128,<%=rb.getString("ZhengXing")%>)</span>
							</div>
							<el-input v-model.trim='gnbImportSaveForm.prefixLength'></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("WangGuan")%>' v-if="['Static','Staticv6'].includes(gnbImportSaveForm.addressType )" prop='gateway' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gnbImportSaveForm.gateway' ></el-input>
						</el-form-item>						
						<div v-if="!importAddVlanShow">
							<el-form-item label="VLAN ID" prop='vlanId' style="margin-bottom: 10px; " class="titleAndNoteBox">
								<el-select v-model='gnbImportSaveForm.vlanId' >
									<el-option v-for="item in importVlanIdList" :label='item' :value='item'></el-option>
								</el-select>
							</el-form-item> 
							<div class='commonFlex' style="margin-top: 4px;">
								<span class='commonImportSize14'><%=rb.getString("TianJiaXinDeVlan")%></span>
								<div class="allowMoreInputAddBtnCls" @click="addVlanClick('import')">
									<span class="el-icon el-icon-plus"></span>
									<span><%=rb.getString("TianJia")%></span>
								</div>
							</div>
						</div>
						<div v-if="importAddVlanShow">
							<span class="commonSize14"><%=rb.getString("TianJiaVlan")%></span>
							<div class="specialItemCls" style="margin: 10px 0;padding: 20px 20px 0;">
								<el-form-item prop="vlanName" class="titleAndNoteBox commonParamImportItem">
									<div slot="label">
										<span class="commonSize14"><%=rb.getString("VLANMingCheng")%></span>
										<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("GNBChangDuTiShi")%>1~13,<%=rb.getString("ZiFuFuShu")%>)</span>
									</div>
									<el-input v-model.trim='gnbImportSaveForm.vlanName' maxlength="13"></el-input>
								</el-form-item>
								<el-form-item prop="vlanId" class="titleAndNoteBox commonParamImportItem">
									<div slot="label">
										<span class="commonSize14">VLAN ID</span>
										<span class="commonNotes12" style="margin-left: 5px;">(<%=rb.getString("FanWei")%>：2~4094,<%=rb.getString("ZhengXing")%>)</span>
									</div>
									<el-input v-model.trim='gnbImportSaveForm.vlanId'></el-input>
								</el-form-item>
								<div class="specialItemDelIcon">
									<span class="el-icon el-icon-circle-close" @click="closeAddVlanClick('import')"></span>
								</div>
							</div>
						</div>
					</div>
					<div v-if="commonImportOperModel == 'importLan'" style='display:flex;margin-left:16px;flex-wrap: wrap'>
						<el-form-item label='<%=rb.getString("IPLeiXing")%>' prop='addressType' class="titleAndNoteBox commonParamImportItem" v-if='false'>
							<el-select v-model='gnbImportSaveForm.addressType'>
								<el-option label='Static' value='Static'></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label='<%=rb.getString("IPDiZhi")%>' prop='ipAddress' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gnbImportSaveForm.ipAddress'></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("GNBZiWangYanMa")%>' prop='subnetMask' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gnbImportSaveForm.subnetMask'></el-input>
						</el-form-item>
					</div>
				</el-form>
			</div>
			<div slot="footer" class="importFooter">
				<el-button type="primary" @click="paramsImportSaveDialogSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="paramsImportSaveDialogClose"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</el-dialog><!--batch import: view -->
		<div class='rightBox infoSpecifiedDevice' style='position: relative;' v-show='commonImportFileInfoShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14' style='font-size: 16px;'>SN:{{importFileInfoSN}}</span>
   				<span class='closeIconBox' @click='commonImportFileInfoCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent' style='padding: 0 20px;'>
                <span class=' commonSize14'><%=rb.getString("GNBMingCheng")%>:{{importFileInfoCellName}}</span>
   				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'>Parameters configuration planning information.</span></div>   				
   				<div style='height: calc(100% - 110px);overflow-y: scroll;'>
   					<el-collapse v-model="infoActiveName">
                        <el-collapse-item name="gnb">
                            <template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonGeneralBold12'>GNB</span></p></template>
                            <div class='commonBorderBottom'>
                                <div class='commonText' style='padding-top: 0;'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBMingCheng")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.gnbName}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBBiaoShiChangDu")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.gnbLength}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBBiaoShi")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.gnbId}}</span>
                                </div>
                            </div>
                        </el-collapse-item>
   					</el-collapse>
					<el-collapse>
                    	<el-collapse-item>
                            <template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonGeneralBold12'>CELL</span></p></template>
                            <div class='commonBorderBottom'>
                                <div class='commonText' style='padding-top: 0;'>
                                    <span class='commonTitleText12'>PCI</span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.pci}}</span>
                                </div>
								<div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("SSBPinDianHao")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.ssbFrequency}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("XiaXingNRARFCN")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.nrarfcndl}}</span>
                                </div>
								<div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("ShangXingNRARFCN")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.nrarfcnul}}</span>
                                </div>
								<div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBPinDaiZhiShi")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.freqBandIndicator}}</span>
                                </div>
								<div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBXiaXingDaiKuan")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.dlbandwidth}}MHz</span>
                                </div>
								<div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBShangXingDaiKuan")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.ulbandwidth}}MHz</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("JiZhanZhiShi")%></span>
									<span class='commonGeneral12 marginRight5' v-if='paramsImportInfo.duplex_mode == "TDDMode"'>TDD</span>
                                    <span class='commonGeneral12 marginRight5' v-else>FDD</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBXiaXingZiZaiBoJianGe")%></span>
									<span class='commonGeneral12 marginRight5' v-if='paramsImportInfo.dlSubcarrierSpacing == "1"'>30kHz</span>
                                    <span class='commonGeneral12 marginRight5' v-else>15kHz</span>
                                </div>
								<div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBShangXingXingZiZaiBoJianGe")%></span>
									<span class='commonGeneral12 marginRight5' v-if='paramsImportInfo.ulSubcarrierSpacing == "1"'>30kHz</span>
                                    <span class='commonGeneral12 marginRight5' v-else>15kHz</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBFaSheTongDaoShu")%></span><!--上行（发送）是基站的接收，移动端的发射-->
									<span class='commonGeneral12 marginRight5'>{{paramsImportInfo.ulAntNum}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBJieShouTongDaoShu")%></span><!--下行：接收-->
									<span class='commonGeneral12 marginRight5'>{{paramsImportInfo.dlAntNum}}</span>
                                </div>
								<div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("MoShi1ChunShuZhouQi")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.dlulTransmissionPeriodicity1}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("MoShi1LianXuXiaXingXiShu")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.nrofDownlinkSlots1}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("MoShi1LianXuShangXingXiShu")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.nrofUplinkSlots1}}</span>
                                </div>
								<div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("MoShi1LianXuXiaXingFuHaoShu")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.nrofDownlinkSymbols1}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("MoShi1LianXuShangXingFuHaoShu")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.nrofUplinkSymbols1}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("MoShi2ChunShuZhouQi")%></span>
									<span class='commonGeneral12 marginRight5' v-if='paramsImportInfo.dlulTransmissionPeriodicity2 == "4095"'><%=rb.getString("Guan")%></span>
                                    <span class='commonGeneral12 marginRight5' v-else>{{paramsImportInfo.dlulTransmissionPeriodicity2}}</span>
                                </div>
								<div class='commonText' v-if="paramsImportInfo.dlulTransmissionPeriodicity2 != '4095'">
                                    <span class='commonTitleText12'><%=rb.getString("MoShi2LianXuXiaXingXiShu")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.nrofDownlinkSlots2}}</span>
                                </div>
                                <div class='commonText' v-if="paramsImportInfo.dlulTransmissionPeriodicity2 != '4095'">
                                    <span class='commonTitleText12'><%=rb.getString("MoShi2LianXuShangXingXiShu")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.nrofUplinkSlots2}}</span>
                                </div>
                                <div class='commonText' v-if="paramsImportInfo.dlulTransmissionPeriodicity2 != '4095'">
                                    <span class='commonTitleText12'><%=rb.getString("MoShi2LianXuXiaXingFuHaoShu")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.nrofDownlinkSymbols2}}</span>
                                </div>
								<div class='commonText' v-if="paramsImportInfo.dlulTransmissionPeriodicity2 != '4095'">
                                    <span class='commonTitleText12'><%=rb.getString("MoShi2LianXuShangXingFuHaoShu")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.nrofUplinkSymbols2}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBGenXuLieSuoYin")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.prachRootSequenceIndex}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBGenXuLieZhi")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.prachRootSequenceValue}}</span>
                                </div>
                            </div>
                        </el-collapse-item>
   					</el-collapse>
					<el-collapse>
                        <el-collapse-item>
                            <template slot='title'> <p style="display:inline-block;margin-left: 26px;"><span class='commonGeneralBold12'>PLMN</span></p></template>
                            <div class='commonBorderBottom'>
                                <div style='padding-top: 5px; margin-bottom: 10px; background: #FCFCFC;border: 1px solid #D5DCEC; border-radius: 5px;' v-for='item in paramsImportInfo.plmn'>
									<div class='commonTextPlmn' style='padding: 6px 10px 3px;'>
										<span class='commonTitleText12'>ID:</span>
										<span class='commonGeneral12'>{{item.index}}</span>
									</div>
									<div class='commonTextPlmn' style='padding: 3px 10px;'>
										<span class='commonTitleText12'><%=rb.getString("GNBNCI")%>:</span>
										<span class='commonGeneral12'>{{item.NCI}}</span>
									</div>
								 	<div class='commonTextPlmn' style='padding: 3px 10px 6px;'>
										<span class='commonTitleText12'><%=rb.getString("GNBGenZongQuYuMa")%>:</span>
										<span class='commonGeneral12'>{{item.tac}}</span>
								 	</div>
									 <div class='commonTextPlmn' style='padding: 3px 10px 6px;'>
										<span class='commonTitleText12'><%=rb.getString("GNBWuXianJieRuWangQuYuMa")%>:</span>
										<span class='commonGeneral12'>{{item.ranac}}</span>
								 	</div>
									<div style='margin: 0 20px;' v-for='items in item.plmnConfigList'>
										<span class='commonTitleText12'><%=rb.getString("GNBPLMNSheZhi")%></span>
										<div style='padding-top: 5px; margin-bottom: 10px; background: #FCFCFC;border: 1px solid #D5DCEC; border-radius: 5px;'>
											<div class='commonTextPlmn' style='padding: 6px 10px 3px;'>
												<span class='commonTitleText12'>ID:</span>
												<span class='commonGeneral12'>{{items.index}}</span>
											</div>
											<div class='commonTextPlmn' style='padding: 3px 10px;'>
												<span class='commonTitleText12'><%=rb.getString("GNBPLMNBiaoShi")%>:</span>
												<span class='commonGeneral12'>{{items.plmnId}}</span>
											</div>
											 <div class='commonTextPlmn' style='padding: 3px 10px 6px;'>
												<span class='commonTitleText12'><%=rb.getString("GNBZhuPLMN")%>:</span>
												<span class='commonGeneral12'>{{items.primary}}</span>
											 </div>
											 <div style='margin: 0 20px;' v-for='itemSlice in items.sliceList'>
												<span class='commonTitleText12'><%=rb.getString("GNBQiePianLieBiao")%></span>
												<div style='padding-top: 5px; margin-bottom: 10px; background: #FCFCFC;border: 1px solid #D5DCEC; border-radius: 5px;'>
													<div class='commonTextPlmn' style='padding: 6px 10px 3px;'>
														<span class='commonTitleText12'>SD:</span>
														<span class='commonGeneral12' v-if='itemSlice.sd == "0"'><%=rb.getString("GNBKong")%></span>
														<span class='commonGeneral12' v-else><%=rb.getString("GNBFeiKong")%></span>
													</div>
													<div class='commonTextPlmn' style='padding: 3px 10px;'>
														<span class='commonTitleText12'>SNSSAI:</span>
														<span class='commonGeneral12'>{{itemSlice.sd_value}}</span>
													</div>												
												</div>
											</div>
										</div>
									</div>
								</div>
                            </div>
                        </el-collapse-item>
   					</el-collapse>
					<el-collapse>
                        <el-collapse-item>
                            <template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonGeneralBold12'>AMF</span></p></template>
                            <div class='commonBorderBottom'>
                                <div class='commonBorder' style='padding-top: 5px; margin-bottom: 5px; background: #FCFCFC;' v-for='item in paramsImportInfo.amf'>
									<div class='commonText1' style='padding: 6px 10px 3px;'>
										<span class='commonTitleText12'>AMF IP</span>
										<span class='commonGeneral12'>{{item.amfIp}}</span>
									</div>								 
									<div class='commonText1' style='padding: 3px 10px;'>
										<span class='commonTitleText12'><%=rb.getString("GNBPLMNBiaoShi")%></span>
										<span class='commonGeneral12'>{{item.plmnId}}</span>
									</div>
								 	<div class='commonText1' style='padding: 3px 10px 6px;'>
										<span class='commonTitleText12'><%=rb.getString("APNMoRen")%></span>
										<span class='commonGeneral12'>{{item.default}}</span>
								 	</div>
								</div>
                            </div>
                        </el-collapse-item>
   					</el-collapse>
					<el-collapse>
                        <el-collapse-item>
                            <template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonGeneralBold12'><%=rb.getString("JieKouSheZhi")%></span></p></template>
                            <div class='commonBorderBottom'>
								<div class="commonGeneralBold12"><%=rb.getString("WANLANlieBiao")%></div>
                                <div class='commonBorder' style='padding-top: 5px; margin-bottom: 5px;background: #FCFCFC;' v-for='item in paramsImportInfo.wan'>
									<div class='commonText1' style='padding: 6px 10px 3px;'>
										<span class='commonTitleText12'><%=rb.getString("IPLeiXing")%></span>
										<span class='commonGeneral12' v-if='item.addressType =="DHCP"'>DHCP</span>
										<span class='commonGeneral12' v-if='item.addressType =="Static"'>Static</span>
										<span class='commonGeneral12' v-if='item.addressType =="DHCPv6"'>IPv6 DHCP</span>
										<span class='commonGeneral12' v-if='item.addressType =="Staticv6"'>IPv6 Static</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px;'>
										<span class='commonTitleText12'><%=rb.getString("IPDiZhi")%></span>
										<span class='commonGeneral12'>{{item.ipAddress}}</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px 6px;'>
										<span class='commonTitleText12'><%=rb.getString("QianZhuiChangDuZiWangYanMa")%></span>
										<span class='commonGeneral12' v-if='item.addressType =="Static"'>{{item.subnetMask}}</span>
										<span class='commonGeneral12' v-else-if='item.addressType =="Staticv6"'>{{item.prefixLength}}</span>
										<span class='commonGeneral12' v-else></span>
									</div>
									<div class='commonText1' style='padding: 3px 10px;'>
										<span class='commonTitleText12'><%=rb.getString("WangGuan")%></span>
										<span class='commonGeneral12'>{{item.gateway}}</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px 6px;'>
										<span class='commonTitleText12'><%=rb.getString("ChengZaiLeiXing")%></span>
										<span class='commonGeneral12'>{{item.bearType}}</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px;'>
										<span class='commonTitleText12'><%=rb.getString("VLANMingCheng")%></span>
										<span class='commonGeneral12'>{{item.vlanName}}</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px 6px;'>
										<span class='commonTitleText12'>VLAN ID</span>
										<span class='commonGeneral12'>{{item.vlanId}}</span>
									</div>
								</div>
								<div class='commonBorderBottom' >
									<div class="commonGeneralBold12"><%=rb.getString("LANLieBiao")%></div>
									<div class='commonBorder' style='padding-top: 5px; margin-bottom: 5px;background: #FCFCFC;' v-for='item in paramsImportInfo.lan'>
										<div class='commonText1' style='padding: 3px 10px;'>
											<span class='commonTitleText12'><%=rb.getString("IPDiZhi")%></span>
											<span class='commonGeneral12'>{{item.ipAddress}}</span>
										</div>
										<div class='commonText1' style='padding: 3px 10px 6px;'>
											<span class='commonTitleText12'><%=rb.getString("GNBZiWangYanMa")%></span>
											<span class='commonGeneral12'>{{item.subnetMask}}</span>
										</div>
									</div>
								</div>
                            </div>
                        </el-collapse-item>
   					</el-collapse>
					<el-collapse>
                        <el-collapse-item>
                            <template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonGeneralBold12'><%=rb.getString("GuanLiFuWu")%></span></p> </template>
                            <div class='commonBorderBottom'>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("DingQiTongZhiKaiGuan")%></span>
                                    <span class='commonGeneral12 marginRight5' v-if='paramsImportInfo.periodicInformEnable == "1"'><%=rb.getString("KeYong")%></span>
									<span class='commonGeneral12 marginRight5' v-else><%=rb.getString("BuKeYong")%></span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("DingQiTongZhiShiJian")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.periodicInformTime}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("DingQiTongZhiJianGe")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.periodicInformInterval}}</span>
                                </div>
                            </div>
                        </el-collapse-item>
   					</el-collapse>
					<el-collapse>
                        <el-collapse-item>
                            <template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonGeneralBold12'>NTP</span></p></template>
                            <div class='commonBorderBottom'>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("NTPKaiGuan")%></span>
									<span class='commonGeneral12 marginRight5' v-if='paramsImportInfo.ntpEnable == "1"'>Enable</span>
									<span class='commonGeneral12 marginRight5' v-else><%=rb.getString("BuKeYong")%></span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("NTPFuWuQi")%> 1</span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.ntpServer1}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("NTPFuWuQi")%> 2</span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.ntpServer2}}</span>
                                </div>
								<div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("NTPFuWuQi")%> 3</span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.ntpServer3}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("NTPFuWuQi")%> 4</span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.ntpServer4}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("NTPFuWuQi")%> 5</span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.ntpServer5}}</span>
                                </div>
								<div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GNBShiQu")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{paramsImportInfo.localTimeZone}}</span>
                                </div>
                            </div>
                        </el-collapse-item>
   					</el-collapse>
				</div>
   			</div>
   		</div>
	</div>
</div>
<script>
    var gnbAddOrEditConfigVue = new Vue({
        el: '#gnbAddOrEditConfigPage',
        data() {
            var vm = this,
	            validatePolicyName = function(rule,value,callback){//basic info：policy name
					if(value === '' || value === null || value === undefined) {
						callback(new Error('<%=rb.getString("BiTian")%>'));
					}else{
						callback();
					}
				},
				validateVersion = function(rule, value, callback) {//upgrade: original version
					var verType = vm.specifyVersionType;
					if(vm.gnbAddOrEditForm.upgradeEnable == '1' && vm.gnbIsReadOnly == false){
	                    if(verType == '1') {
	                    	callback();
	                    }else if(vm.gnbResultOriginalVersionList.length == 0) {
	                    	callback('<%=rb.getString("QingXuanZeJiLu") %>');
	                    }else{
	                    	callback();
	                    }
                    }else{
                    	callback();
                    }
                },
                validateTarget = function(rule,value,callback) {//upgrade: target version
					if(vm.gnbAddOrEditForm.upgradeEnable == '1' && vm.gnbIsReadOnly == false){
	                	if( value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("BiTian")%>');
						}else {
							callback();
						}
					}else{
                    	callback();
                    }
				},
				validateFilesName = function(rule,value,callback) {//license import: file name
	            	value = vm.fileName;
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else {
						callback();
					}
				},
				validateFileNames = (rule,value,callback) => {//batch import: file name
					var value = vm.fileName; 
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else if(!fileFormatMatch(value,"xlsx,xls")){
						callback(new Error('<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>'));
					}else {
						callback();
					} 
				},
				validatePlmn = (rule,value,callback)=>{//-------------------------------------- params config、batch import: common validate
					var errorTip = rule.errorMsg, isRequired = false; 
					if(vm.gnbAddOrEditForm.selfConfigEnable == '1' && vm.gnbAddOrEditForm.paramConfigEnable == '1' && vm.gnbIsReadOnly == false){
						isRequired = true;
					}else{ isRequired = false; }
					if(isRequired == true){
						if(vm.gnbAddOrEditForm.plmn.length == 0){
							callback(new Error(errorTip))
						}else{
							callback();
						}
					}else{
						callback();
					}
				},
				validateImportPlmn = (rule,value,callback)=>{//batch import: plmn
					var errorTip = rule.errorMsg;
					if(vm.gnbImportAddOrEditForm.plmn == 0){
						callback(new Error(errorTip))
					}else{
						callback();
					}
				},
				validateRange = (rule,value,callback)=>{
					var min = rule.min, max = rule.max, errorTip = rule.errorMsg, isRequired = false,  reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/; 
					if(vm.gnbAddOrEditForm.selfConfigEnable == '1' && vm.gnbAddOrEditForm.paramConfigEnable == '1' && vm.gnbIsReadOnly == false){
						isRequired = true;
					}else{ isRequired = false; }
					if(isRequired == true){
						if(value === '' || value === null || value === undefined){
							callback(new Error(errorTip))
						}else{
							if(reg.test(value)){
								if(value < min || value > max){
									callback(new Error(errorTip))
								}else{
									callback();
								}
							}else{
								callback(new Error(errorTip))
							}
						}
					}else{
						if(value === '' || value === null || value === undefined){
							callback();
						}else{
							if(reg.test(value)){
								if(value < min || value > max){
									callback(new Error(errorTip))
								}else{
									callback();
								}
							}else{
								callback(new Error(errorTip))
							}
						}
					}
				},
				validateGnbLengthRange = (rule,value,callback)=>{
					var min = rule.min, max = rule.max, errorTip = rule.errorMsg, reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/; 
					if(value === '' || value === null || value === undefined){
						callback(new Error(errorTip))
					}else{
						if(reg.test(value)){
							if(value < min || value > max){
								callback(new Error(errorTip))
							}else{
								callback();
							}
						}else{
							callback(new Error(errorTip))
						}
					}
				},
				validatePrachValueRange = (rule,value,callback)=>{
					var max = '', errorTip = '', reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
					if(vm.gnbAddOrEditForm.prachRootSequenceIndex == '0'){
						max = 837; errorTip = '<%=rb.getString("FanWei")%>：0~837,<%=rb.getString("ZhengXing")%>';
					}else{
						max = 137; errorTip = '<%=rb.getString("FanWei")%>：0~137,<%=rb.getString("ZhengXing")%>';
					}					
					if(value === '' || value === null || value === undefined){
						callback();
					}else{
						if(reg.test(value) && parseInt(value)>=0 && parseInt(value)<=max) {
							callback();
						}else {
							callback(errorTip);
						}
					}
				},
				validateImportPrachValueRange = (rule,value,callback)=>{
					var max = '', errorTip = '', reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
					if(vm.gnbImportAddOrEditForm.prachRootSequenceIndex == '0'){
						max = 837; errorTip = '<%=rb.getString("FanWei")%>：0~837,<%=rb.getString("ZhengXing")%>';
					}else{
						max = 137; errorTip = '<%=rb.getString("FanWei")%>：0~137,<%=rb.getString("ZhengXing")%>';
					}					
					if(value === '' || value === null || value === undefined){
						callback();
					}else{
						if(reg.test(value) && parseInt(value)>=0 && parseInt(value)<=max) {
							callback();
						}else {
							callback(errorTip);
						}
					}
				},
				validateNotRequireRange = (rule,value,callback)=>{
					var min = rule.min, max = rule.max, errorTip = rule.errorMsg, reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/; 
					if(value === '' || value === null || value === undefined){
						callback();
					}else{
						if(reg.test(value)){
							if(value < min || value > max){
								callback(new Error(errorTip))
							}else{
								callback();
							}
						}else{
							callback(new Error(errorTip))
						}
					}
				},
				commonPeriodTime = (rule,value,callback)=>{
					var isRequired = false;
					if(vm.advanceTreeSelected.includes('MANAGEMENT') || vm.advanceTreeSelected.includes('NTP')){
						isRequired = true;
					}else{
						isRequired = false;
					}
					if(isRequired == true){
						if(value === '' || value === null || value === undefined){
							callback('<%=rb.getString("BiTian")%>');
						}else{
							callback();
						}
					}else{
						callback();
					}
				},
				commonImportPeriodTime = (rule,value,callback)=>{
					if(value === '' || value === null || value === undefined){
						callback('<%=rb.getString("BiTian")%>');
					}else{
						callback();
					}
				},
				validatePeriodRange  = (rule,value,callback)=>{
					var min = rule.min, errorTip = rule.errorMsg, isRequired = false, reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/; 
					if(vm.advanceTreeSelected.includes('MANAGEMENT') || vm.advanceTreeSelected.includes('NTP')){
						isRequired = true;
					}else{
						isRequired = false;
					}
					if(isRequired == true){
						if(value === '' || value === null || value === undefined){
							callback(new Error(errorTip))
						}else{
							if(reg.test(value)){
								if(value < min){ // 小于 0
									callback(new Error(errorTip))
								}else{
									callback();
								}
							}else{
								callback(new Error(errorTip))// 不是数字
							}
						}
					}else{
						if(value === '' || value === null || value === undefined){
							callback();
						}else{
							if(reg.test(value)){
								if(value < min){
									callback(new Error(errorTip))
								}else{
									callback();
								}
							}else{
								callback(new Error(errorTip))
							}
						}
					}
				},
				validateImportPeriodRange = (rule,value,callback)=>{
					var min = rule.min, errorTip = rule.errorMsg, reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/; 
					if(value === '' || value === null || value === undefined){
						callback(new Error(errorTip))
					}else{
						if(reg.test(value)){
							if(value < min){ // 小于 0
								callback(new Error(errorTip))
							}else{
								callback();
							}
						}else{
							callback(new Error(errorTip))// 不是数字
						}
					}
				},
				validateKeyOpc = (rule,value,callback) => {
					var reg = /^[0-9a-zA-Z]{2}([0-9a-zA-Z]{2})*$/;
					if(value === '' || value === null || value === undefined){
						callback();
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("CuoWu")%>'))
						}
					}
				},
				validateGnbIdRange = (rule,value,callback) => {
					var min = rule.min, max = rule.max, errorTip = rule.errorMsg, isRequired = false;
					if(vm.gnbAddOrEditForm.selfConfigEnable == '1' && vm.gnbAddOrEditForm.paramConfigEnable == '1' && vm.gnbIsReadOnly == false){
						isRequired = true;//vm.gnbAddOrEditForm. selfConfigEnable 和.vm.gnbAddOrEditForm.paramConfigEnable 都打开时， 校验必填项;
					}else{
						isRequired = false;
					}
					if(isRequired == true){//支持配置范围，例如： 1..100,配置多个范围时，以分号分隔， 例如1..100; 200..1000; 
						if(value === '' || value === null || value === undefined){
							callback(new Error(errorTip))
						}else{
							if(value.indexOf(';') != -1){//判断value 是否包含分号
								var arr = value.split(';'),
									flag = true;
								for(var i=0;i<arr.length;i++){
									var item = arr[i];
									if(item.indexOf('..') != -1){
										var itemArr = item.split('..');
										if(itemArr.length == 2){
											if(itemArr[0] == '' || itemArr[1] == ''){
												flag = false;
												break;
											}else{
												if(vm.isNumeric(itemArr[0]) && vm.isNumeric(itemArr[1])){
													if(itemArr[0] < min || itemArr[1] > max){
														flag = false;
														break;
													}
												}else{
													flag = false;
													break;
												}
											}
										}else{
											flag = false;
											break;
										}
									}else{
										if(item == ''){
											flag = false;
											break;
										}else{
											if(vm.isNumeric(item)){
												if(item < min || item > max){
													flag = false;
													break;
												}
											}else{
												flag = false;
												break;
											}
										}
									}
								}
								if(flag){
									callback();
								}else{
									callback(new Error(errorTip))
								}
							}else{
								if(value.indexOf('..') != -1){
									var itemArr = value.split('..');
									if(itemArr.length == 2){
										if(itemArr[0] == '' || itemArr[1] == ''){
											callback(new Error(errorTip))
										}else{
											if(vm.isNumeric(itemArr[0]) && vm.isNumeric(itemArr[1])){
												if(itemArr[0] < min || itemArr[1] > max){
													callback(new Error(errorTip))
												}else{
													callback();
												}
											}else{
												callback(new Error(errorTip))
											}
										}
									}else{
										callback(new Error(errorTip))
									}
								}else{
									if(value == ''){
										callback(new Error(errorTip))
									}else{
										if(vm.isNumeric(value)){
											if(value < min || value > max){
												callback(new Error(errorTip))
											}else{
												callback();
											}
										}else{
											callback(new Error(errorTip))
										}
									}
								}
							}
						}
					}else{
						if(value === '' || value === null || value === undefined){
							callback();
						}else{
							if(value.indexOf(';') != -1){//判断value 是否包含分号
								var arr = value.split(';'),
									flag = true;
								for(var i=0;i<arr.length;i++){
									var item = arr[i];
									if(item.indexOf('..') != -1){
										var itemArr = item.split('..');
										if(itemArr.length == 2){
											if(itemArr[0] == '' || itemArr[1] == ''){
												flag = false;
												break;
											}else{
												if(vm.isNumeric(itemArr[0]) && vm.isNumeric(itemArr[1])){
													if(itemArr[0] < min || itemArr[1] > max){
														flag = false;
														break;
													}
												}else{
													flag = false;
													break;
												}
											}
										}else{
											flag = false;
											break;
										}
									}else{
										if(item == ''){
											flag = false;
											break;
										}else{
											if(vm.isNumeric(item)){
												if(item < min || item > max){
													flag = false;
													break;
												}
											}else{
												flag = false;
												break;
											}
										}
									}
								}
								if(flag){
									callback();
								}else{
									callback(new Error(errorTip))
								}
							}else{
								if(value.indexOf('..') != -1){
									var itemArr = value.split('..');
									if(itemArr.length == 2){
										if(itemArr[0] == '' || itemArr[1] == ''){
											callback(new Error(errorTip))
										}else{
											if(vm.isNumeric(itemArr[0]) && vm.isNumeric(itemArr[1])){
												if(itemArr[0] < min || itemArr[1] > max){
													callback(new Error(errorTip))
												}else{
													callback();
												}
											}else{
												callback(new Error(errorTip))
											}
										}
									}else{
										callback(new Error(errorTip))
									}
								}else{
									if(value == ''){
										callback(new Error(errorTip))
									}else{
										if(vm.isNumeric(value)){
											if(value < min || value > max){
												callback(new Error(errorTip))
											}else{
												callback();
											}
										}else{
											callback(new Error(errorTip))
										}
									}
								}
							}
						}
					}
				},
				validateImportGnbIdRange = (rule,value,callback) => {  //支持配置范围，例如： 1..100,配置多个范围时，以分号分隔， 例如1..100; 200..1000; 
					var min = rule.min, max = rule.max, errorTip = rule.errorMsg;
					if(value === '' || value === null || value === undefined){
						callback(new Error(errorTip))
					}else{
						if(value.indexOf(';') != -1){//判断value 是否包含分号
							var arr = value.split(';'),
								flag = true;
							for(var i=0;i<arr.length;i++){
								var item = arr[i];
								if(item.indexOf('..') != -1){
									var itemArr = item.split('..');
									if(itemArr.length == 2){
										if(itemArr[0] == '' || itemArr[1] == ''){
											flag = false;
											break;
										}else{
											if(vm.isNumeric(itemArr[0]) && vm.isNumeric(itemArr[1])){
												if(itemArr[0] < min || itemArr[1] > max){
													flag = false;
													break;
												}
											}else{
												flag = false;
												break;
											}
										}
									}else{
										flag = false;
										break;
									}
								}else{
									if(item == ''){
										flag = false;
										break;
									}else{
										if(vm.isNumeric(item)){
											if(item < min || item > max){
												flag = false;
												break;
											}
										}else{
											flag = false;
											break;
										}
									}
								}
							}
							if(flag){
								callback();
							}else{
								callback(new Error(errorTip))
							}
						}else{
							if(value.indexOf('..') != -1){
								var itemArr = value.split('..');
								if(itemArr.length == 2){
									if(itemArr[0] == '' || itemArr[1] == ''){
										callback(new Error(errorTip))
									}else{
										if(vm.isNumeric(itemArr[0]) && vm.isNumeric(itemArr[1])){
											if(itemArr[0] < min || itemArr[1] > max){
												callback(new Error(errorTip))
											}else{
												callback();
											}
										}else{
											callback(new Error(errorTip))
										}
									}
								}else{
									callback(new Error(errorTip))
								}
							}else{
								if(value == ''){
									callback(new Error(errorTip))
								}else{
									if(vm.isNumeric(value)){
										if(value < min || value > max){
											callback(new Error(errorTip))
										}else{
											callback();
										}
									}else{
										callback(new Error(errorTip))
									}
								}
							}
						}
					}
				},
				validateAmfIpAddress= (rule,value,callback) => {
					if(value === ''){
						callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
					}else{
						if(vm.isValidIP(value) || vm.isIPv6(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("IPGeShiBuDui")%>'))
						}
					}
				},
				validateIpsecGateway = (rule,value,callback) => {
					if(value == '' || value == undefined || value == null){
						callback(new Error('<%=rb.getString("FanWei")%>：1~64,Digit string'))
					}else{
						callback();
					}
				},
				validateIPSecKeyLeft = (rule,value,callback) => {
					var typeVal = vm.addIPSecDialogForm.key_lefe_type,
						codes={
							's':31536000,
							'm':525600,
							'h':8760,
							'd':365
						}
						minNum = 1, maxNum = codes[typeVal];
					if(value == '' || value == undefined || value == null){
						callback('<%=rb.getString("CuoWu")%>');
					}else{
						if(vm.isNumeric(value) && parseInt(value)>=minNum && parseInt(value)<=maxNum){
							callback();
						}else{
							callback(new Error('<%=rb.getString("CuoWu")%>'))
						}
					}
				},
				validateIPSecIKELeftTime = (rule,value,callback) => {
					var typeVal = vm.addIPSecDialogForm.ike_lefe_time_type,
						codes={
							's':31536000,
							'm':525600,
							'h':8760,
							'd':365
						}
						minNum = 1, maxNum = codes[typeVal];
					if(value == '' || value == undefined || value == null){
						callback('<%=rb.getString("CuoWu")%>');
					}else{
						if(vm.isNumeric(value) && parseInt(value)>=minNum && parseInt(value)<=maxNum){
							callback();
						}else{
							callback(new Error('<%=rb.getString("CuoWu")%>'))
						}
					}
				},
				validateIPSecRekeyMargin = (rule,value,callback) => {
					var typeVal = vm.addIPSecDialogForm.rekey_margin_type,
						codes={
							's':31536000,
							'm':525600,
							'h':8760,
							'd':365
						}
						minNum = 1, maxNum = codes[typeVal];
					if(value == '' || value == undefined || value == null){
						callback('<%=rb.getString("CuoWu")%>');
					}else{
						if(vm.isNumeric(value)&&parseInt(value)>=minNum && parseInt(value)<=maxNum){
							callback();
						}else{
							callback(new Error('<%=rb.getString("CuoWu")%>'))
						}
					}
				},
				validateIPSecDpddelay = (rule,value,callback) => {
					var typeVal = vm.addIPSecDialogForm.dpddelay_type,
						codes={
							's':31536000,
							'm':525600,
							'h':8760,
							'd':365
						}
						minNum = 1, maxNum = codes[typeVal];
					if(value == '' || value == undefined || value == null){
						callback('<%=rb.getString("CuoWu")%>');
					}else{
						if(vm.isNumeric(value) && parseInt(value)>=minNum && parseInt(value)<=maxNum){
							callback();
						}else{
							callback(new Error('<%=rb.getString("CuoWu")%>'))
						}
					}
				},
				validateCommonRange = (rule,value,callback)=>{//-------------------------------------------- advance  右侧窗口 form 校验
					var min = rule.min, max = rule.max, errorTip = rule.errorMsg, reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/; 
					if(value === '' || value === null || value === undefined){
						callback(new Error(errorTip))
					}else{
						if(reg.test(value)){
							if(value < min || value > max){
								callback(new Error(errorTip))
							}else{
								callback();
							}
						}else{
							callback(new Error(errorTip))
						}
					}	
				},
				validatePlmnId = (rule,value,callback) => {
					var reg = /^[0-9]{5,6}$/;
					if(value === ''){
						callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
					}else{
						if(reg.test(value)){
							callback();
						}else{
							callback(new Error('Length<%=rb.getString("MaoHao")%> 5~6 Digit <%=rb.getString("ZhengXing")%>'))
						}
					}
				},
				validateDestinationNetwork = (rule,value,callback) => {
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}else if(vm.isValidIP(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}
				},
				validateNetmask = (rule,value,callback) => {
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
					}else if(vm.isMask(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
					}
				},
				validateGateway = (rule,value,callback) => {
					if(vm.commonAdvanceOperModel == 'router'){
						if(value == '' || value == undefined || value == null){
							callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
						}else if(vm.isValidIP(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
						}
					}else if(vm.commonAdvanceOperModel == 'wan' || vm.commonImportOperModel == 'importWan'){
						if(vm.commonAddEditForm.addressType == 'Static' || vm.gnbImportSaveForm.addressType == 'Static'){
							if(vm.isValidIP(value)){
								callback();
							}else{
								callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
							}
						}else if(vm.commonAddEditForm.addressType == 'Staticv6' || vm.gnbImportSaveForm.addressType == 'Staticv6'){
							if(vm.isIPv6(value)){
								callback();
							}else{
								callback(new Error('<%=rb.getString("QingShuRuHeFaDeIPV6DiZhi")%>'))
							}
						}else{
							callback();
						}
					}	
				},
				validateWanVlanName = (rule,value,callback) => {
					if((vm.addVlanShow && vm.commonAdvanceOperModel == 'wan') || (vm.importAddVlanShow && vm.commonImportOperModel == 'importWan')){
						if(value === '' || value === null || value === undefined){
							callback(new Error('<%=rb.getString("GNBChangDuTiShi")%> 1~13,<%=rb.getString("ZiFuFuShu")%>'))
						}else{
							callback();
						}
					}else{
						callback();
					}
				},
				validateWanVlanID= (rule,value,callback) => {
					if((vm.addVlanShow && vm.commonAdvanceOperModel == 'wan') || (vm.importAddVlanShow && vm.commonImportOperModel == 'importWan')){
						if(value === '' || value === null || value === undefined){
							callback(new Error('<%=rb.getString("FanWei")%>：2~4094,<%=rb.getString("ZhengXing")%>'))
						}else{
							if(vm.isNumeric(value)&&parseInt(value)>=2 && parseInt(value)<=4094){
								callback();
							}else{
								callback(new Error('<%=rb.getString("FanWei")%>：2~4094,<%=rb.getString("ZhengXing")%>'))
							}
						}
					}else{
						callback();
					}			
				},
				validateWanIpAddress = (rule,value,callback) => {
					if(vm.commonAddEditForm.addressType == 'Static'){
						if(vm.isValidIP(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
						}
					}else if(vm.commonAddEditForm.addressType == 'Staticv6' && vm.commonAdvanceOperModel == 'wan'){
						if(vm.isIPv6(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
						}
					}else{
						callback();
					}
				},
				validateImportWanIpAddress = (rule,value,callback) => {
					if(vm.gnbImportSaveForm.addressType == 'Static'){
						if(vm.isValidIP(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
						}
					}else if(vm.gnbImportSaveForm.addressType == 'Staticv6' && vm.commonImportOperModel == 'importWan'){
						if(vm.isIPv6(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
						}
					}else{
						callback();
					}
				},
				validateWanSubnetMask = (rule,value,callback) => {
					if(vm.commonAdvanceOperModel == 'lan' || vm.commonImportOperModel == 'importLan'){
						if(vm.isMask(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
						}
					}else{
						if(vm.commonAddEditForm.addressType == 'Static' || vm.gnbImportSaveForm.addressType == 'Static'){
							if(vm.isMask(value)){
								callback();
							}else{
								callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
							}
						}else{
							callback();
						}
					}
				},
				validateWanPrefixLength = (rule,value,callback) => {
					if(vm.commonAddEditForm.addressType == 'Staticv6' || vm.gnbImportSaveForm.addressType == 'Staticv6'){
						if(vm.isNumeric(value)&&parseInt(value)>=0 && parseInt(value)<=128){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%>：0~128,<%=rb.getString("ZhengXing")%>'))
						}
					}else{
						callback();
					}
				}
            return {
				commonImportRow: {},
				gnbPolicyTitle: 'Add Policy',
				height:"100%",
				gnbOperType: '',
				gnbPolicyId: '',
				gnbIsReadOnly: false,
				gnbAddOrEditForm: {
            		policySwitch: '1',//总开关
            		policyName: '',
            		productType: '',
            		executeType: '0',
            		upgradeEnable: '0',//升级 开关
            		targetVersion: '',
            		originalVersion: '',
            		preserveSetting: '1',
            		licenseEnable: '0', // license 开关
					selfConfigEnable: '0', //批量配置-开关
            		paramConfigEnable: '0', //批量配置-手动配置参数开关
					batchConfigEnable: '0', //批量配置-导入-开关
					gnbName: '',
					gnbLength: '',
					gnbId: '',
					pci: '',
					freqBandIndicator:'41',
					nrarfcndl:'515154',
					dlbandwidth:'100',
					ssbFrequency:'506910',
					nrarfcnul:'515154',
					ulbandwidth:'100',
					duplex_mode:'TDDMode',	//TDD  
					dlSubcarrierSpacing: '1',
					ulSubcarrierSpacing: '1',
					ulAntNum:'2',
					dlAntNum:'2',
					dlulTransmissionPeriodicity1:'6', //6
					nrofDownlinkSlots1: '7',
					nrofUplinkSlots1: '2',
					nrofDownlinkSymbols1: '4',
					nrofUplinkSymbols1: '4',
					dlulTransmissionPeriodicity2: '5',
					nrofDownlinkSlots2: '7',
					nrofUplinkSlots2: '2',
					nrofDownlinkSymbols2: '6', 
					nrofUplinkSymbols2: '4',
					prachRootSequenceIndex:'0',
					prachRootSequenceValue:'0', 
					plmn: [],
					periodicInformEnable: '0',//Management Server
					periodicInformTime: '',
					periodicInformInterval: '',
					ntpEnable: '0',
					localTimeZone: '',
					ntpServer1: '',
					ntpServer2: '',
					ntpServer3: '',
					ntpServer4: '',
					ntpServer5: '',
					amf: [],
					wan: [],
					lan: [],
					ipsecEnable: '0',
					ipsecImsi: '',
					ipsecKey: '',
					ipsecOpc: '',
					ipsecUsimEnable: '1',
					ipsecUsimAuthEnable: '0',
					ipsecTunnel:[],
					staticRouting: [],
					halobEnable: '0',
					halobMode: '1',
					custParam: []
				},
				timeZoneList: [],
				commonPlmnNrRows: {},//添加plmn Cofig 数据时，获取到 plmn nr行数据
				commonImportPlmnNrRows: {},
				commonPlmnConfigRows: {},//添加slice Cofig 数据时，获取到 plmnconfig行数据
				commonImportPlmnConfigRows: {},
				editPlmnCongigRows: {},
				editImportPlmnCongigRows: {},
				editSliceCongigRows: {},
				editImportSliceCongigRows: {},
				cellMoreItemShow: false,
				specifyVersionType: '0', //any checkbox
				gnbAddOrEditRule: {
					policyName: [{validator: validatePolicyName}],
					productType: [{validator: validatePolicyName}],
					originalVersion: [{validator: validateVersion}],
					targetVersion: [{validator: validateTarget}],
					plmn: [{required:true, validator: validatePlmn, errorMsg:'<%=rb.getString("BiTian")%>' }],
					gnbLength:[{required:true, validator: validateRange, min:22, max:32, errorMsg:'<%=rb.getString("FanWei")%>：22~32,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					gnbId:[{required:true, validator: validateGnbIdRange, min:0, max:4294967295, errorMsg:'<%=rb.getString("FanWei")%>0~4294967295,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					pci:[{required:true, validator: validateGnbIdRange, min:0, max:1007, errorMsg:'<%=rb.getString("FanWei")%>：0~1007,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					freqBandIndicator:[{ validator: validateRange, min: 1, max: 1024, errorMsg:'<%=rb.getString("FanWei")%>：1~1024,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrarfcndl:[{ validator: validateRange, min: 0, max: 3279165, errorMsg:'<%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					ssbFrequency:[{ validator: validateRange, min: 0, max: 3279165, errorMsg:'<%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrarfcnul:[{ validator: validateRange, min: 0, max: 3279165, errorMsg:'<%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofDownlinkSlots1:[{ validator: validateRange, min: 0, max: 320, errorMsg:'<%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofUplinkSlots1:[{ validator: validateRange, min: 0, max: 320, errorMsg:'<%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofDownlinkSymbols1:[{ validator: validateRange, min: 0, max: 13, errorMsg:'<%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofUplinkSymbols1:[{ validator: validateRange, min: 0, max: 13, errorMsg:'<%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofDownlinkSlots2:[{ validator: validateRange, min: 0, max: 320, errorMsg:'<%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofUplinkSlots2:[{ validator: validateRange, min: 0, max: 320, errorMsg:'<%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofDownlinkSymbols2:[{ validator: validateRange, min: 0, max: 13, errorMsg:'<%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofUplinkSymbols2:[{ validator: validateRange, min: 0, max: 13, errorMsg:'<%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					prachRootSequenceValue:[{ validator: validatePrachValueRange, trigger:'blur'}],
					periodicInformTime:[{ required:true, validator: commonPeriodTime, errorMsg:'requied', trigger:'blur'}],
					periodicInformInterval:[{required:true, validator: validatePeriodRange, min: 1, errorMsg:'<%=rb.getString("FanWei")%>：1~Infinite,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					localTimeZone:[{ required:true, validator: commonPeriodTime, errorMsg:'requied',trigger:'blur'}],
					ntpServer1:[{ required:true, validator: commonPeriodTime, errorMsg:'requied',trigger:'blur'}],
					ipsecKey:[{ validator: validateKeyOpc, errorMsg:'error', trigger:'blur'}],
					ipsecOpc:[{ validator: validateKeyOpc, errorMsg:'error', trigger:'blur'}]
				},
				gnbProductList: [],
				gnbFunctionModulesSelect: '0',
				gnbParameterConfigActive: 'gnbParamsConfig', 
				gnbActiveName: ['gnb', 'cell', 'plmn', 'managementServer', 'ntp','customized'],
            	gnbTarVersionList: [], //software upgrade
                gnbAddVersionShow: false,
                gnbAddVersionBtnShow: true,
                gnbResultOriginalVersionShow: false,
            	gnbSoftwareAddVersionForm: {
            		versionStr: '',
					versionList: [],
					itemTest:'',
					originalVersion: '',
            	},
            	gnbVersionErrorMessage: '',
            	gnbVersionMessage: '',
            	gnbResultOriginalVersionList: [],
            	gnbSoftwareOriginalVersionList: [],
				gnbCurSelectVersionData: [],
                gnbSearchValue: "",
                gnbLicenseImportShow: false, // license
                gnbLicenseUrl: '${ctx}/gnb/pnp/queryGnbPnPLicensePageList.action',
                gnbLicenseQuery: {
					searchText: '',
					productType: '',
					timeZone: timeZone
				},
				gnbImportModeSelection:'moreFile',
	            fileName:'',
	            gnbLicenseForm: {uploadFileUrl: ''},
	            gnbMoreFileParams:{},
	            gnbMoreFileList:[],
	            gnbFileData:[],
				gnbImportLicenseRules: {fileName:[{validator: validateFilesName}]},
	            gnbUploadBtnDisabled: false,
				gnbLicenseLoading: false,
				commonImportFileInfoShow: false,//parameter config
				urlConfigPlan:"",
				importParamsQuery:{
					timeZone: timeZone,
					searchText: "",
				},
				importParamsQueryForm:{searchText:""},
				commonImportFileShow: false,
				commonImportFileLoading: false,
				commonImportFileDisabled: false,
				gnbImportRuleForm: {
					uploadFileUrl: '',
					operType: "0",
				},
				gnbImportRules:{ fileName:[{ validator: validateFileNames}]},	         	          
				fileParams:{},              
				fileName:'',	            					
				showFileTip:false,
				fileList:[],
				filePath:'',
                gnbSaveBtnDisabled: false,
				gnbParamImportSaveBtnDisabled: false,
				gnbAdvanceTreeShow: false,// advance
				advanceTree: [
					{ id: 'AMFSetting', label: 'NR Setting', children: [{ id: 'AMF', label: 'AMF ' }]},
					{ id: 'NETWORK',label: '<%=rb.getString("WangLuoSheZhi")%>',
						children: [{ id: 'WANLAN', label: '<%=rb.getString("JieKouSheZhi")%>' }, { id: 'IPSEC', label: '<%=rb.getString("IPSecSheZhi")%>' }, { id: 'STATICROUTE', label: '<%=rb.getString("WangLuoSheZhi")%>' }, { id: 'HALOB', label: 'HaloB' }]
					},
					{ id: 'BTSSetting', label: '<%=rb.getString("BTSSheZhi")%>', children: [ { id: 'MANAGEMENT', label: '<%=rb.getString("GuanLiFuWu")%>' }]},
					{ id: 'SYSTEM', label: '<%=rb.getString("XiTong")%>', children: [ { id: 'NTP', label: 'NTP' }] }
				],
				advanceTreeSelected: [],
				defaultRreeChecked: [],
				commonAddOrUpdateParamShow: false,
				commonAddOrUpdateBtnDisabled: false,
				commonAddOrUpdateParamTitle: '<%=rb.getString("TianJia")%>',
				commonImportOperType: '',
				commonImportOperModel: '',
				commonAdvanceOperType:'',
				commonAdvanceOperModel:'',
				commonAddEditForm:{
					NCI: '',//PLMN nr
					tac: '',
					ranac: '',
					plmnId:'',//PLMN nr->plmn setting  //相同字段，与 plmn	有一致的 plmnId
					primary:'',
					sd: '0',//PLMN nr->plmn setting-> slice
					sd_value:'',
					amfIp: '',
					default: '0',
					destinationNetwork: '',//router
					netmask: '',
					gateway: '', //相同字段，与 router 有一致的 gateway
					addressType: 'DHCP',//wan
					ipAddress: '',
					subnetMask: '',
					prefixLength: '',
					bearType: 'NG',
					vlanName: '',
					vlanId: '',
					origin: '',
				},
				commonAddEditFormRules:{
					NCI:[{ required:true, validator: validateCommonRange, min: 0, max: 68719476735, errorMsg:'<%=rb.getString("FanWei")%>：0~68719476735,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					tac:[{ required:true, validator: validateCommonRange, min: 0, max: 16777215, errorMsg:'<%=rb.getString("FanWei")%>：0~16777215,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					ranac:[{ required:true, validator: validateCommonRange, min: 0, max: 255, errorMsg:'<%=rb.getString("FanWei")%>：0~255,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					plmnId:[{ required:true, validator: validatePlmnId, errorMsg:'<%=rb.getString("FanWei")%>：5~6 Digit,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					primary:[{ required:true, message:'<%=rb.getString("BiTian")%>'}],
					sd_value:[{ required:true, validator: validateCommonRange, min: 0, max: 16777215, errorMsg:'<%=rb.getString("FanWei")%>：0~16777215,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					destinationNetwork:[{required:true, validator: validateDestinationNetwork, trigger:'blur'}],	
					netmask:[{required:true, validator: validateNetmask, trigger:'blur'}],	
					gateway:[{required: true, validator: validateGateway, trigger:'blur'}],	
					ipAddress:[{required:true, validator: validateWanIpAddress, trigger:'blur'}],	
					subnetMask:[{required:true, validator: validateWanSubnetMask,trigger:'blur'}],	
					prefixLength:[{required:true, validator: validateWanPrefixLength,trigger:'blur'}],
					vlanName:[{ validator: validateWanVlanName, trigger:'blur'}],	
					vlanId:[{ validator: validateWanVlanID, trigger:'blur'}],
					amfIp:[{required:true, validator: validateAmfIpAddress, trigger:'blur'}],	
				},
				addVlanShow:false,//interface setting
				importAddVlanShow:false,
				addIPSecTunnelDialogShow: false, //ipsec
				advanceIpsecOperType:'',
				activeIPSecTunnelCollapse:['Basic','Advance'],
				addIPSecDialogForm:{
					tunnel_enable:'0',
					tunnel_name:'',
					left_auth:'psk',
					right_auth:'psk',
					gateway:'10.10.10.10',
					right_subnet:'0.0.0.0/0',
					right_id:'C=CH,O=strongSwan,CN=server',
					secret_key:'clientKey.der',
					tunnel_left: '%defaultroute',
					right_secretkey: '',
					left_id:'C=CH,O=strongSwan,CN=server',
					left_cert:'',
					left_source_ip:'%config',
					left_subnet:'',
					fragmentation:'yes',
					ike_encryption:'aes128',
					ike_dh_group:'modp1024',
					ike_authentication:'sha256',
					esp_encryption:'aes128',
					esp_dh_group:'modp1024',
					esp_authentication:'sha256',
					key_lefe:'360',
					key_lefe_type:'d', //注意 接口无此参数
					ike_lefe_time:'360',
					ike_lefe_time_type:'d', //注意 接口无此参数
					rekey_margin:'5',
					rekey_margin_type:'m', //注意 接口无此参数
					dpdaction:'restart',
					dpddelay:'30',
					dpddelay_type:'s', //注意 接口无此参数
					left_interface:'None',
					rekey: 'No',
					reauth:'No',
					forceencaps: 'No',
					mobike: 'No',
					used_source_ip: '', //注意页面无此参数
				},
				addIPSecDialogRules:{
					tunnel_name:[{required:true,message:'error',trigger:'blur'}],
					gateway:[{required:true,validator:validateIpsecGateway,trigger:'blur'}],
					right_subnet:[{required:true,message:'error',trigger:'blur'}],
					tunnel_left:[{required:true,message:'error',trigger:'blur'}],
					key_lefe:[{required:true,message:'error',trigger:'blur'},{validator:validateIPSecKeyLeft,trigger:'blur'}],
					ike_lefe_time:[{required:true,message:'error',trigger:'blur'},{validator:validateIPSecIKELeftTime,trigger:'blur'}],
					rekey_margin:[{required:true,message:'error',trigger:'blur'},{validator:validateIPSecRekeyMargin,trigger:'blur'}],
					dpddelay:[{required:true,message:'error',trigger:'blur'},{validator:validateIPSecDpddelay,trigger:'blur'}],
				},
				downloadVisible: false,
				importFileInfoSN: '',
                importFileInfoCellName: '',
				paramsImportInfo:{
					gnbName: '',
					gnbLength: '',
					gnbId: '',
					pci: '',
					freqBandIndicator: '',
					nrarfcndl: '',
					dlbandwidth:'',
					ssbFrequency:'',
					nrarfcnul:'',
					ulbandwidth:'',
					duplex_mode:'',	  
					dlSubcarrierSpacing: '',
					ulSubcarrierSpacing: '',
					ulAntNum:'',
					dlAntNum:'',
					dlulTransmissionPeriodicity1:'', 
					nrofDownlinkSlots1: '',
					nrofUplinkSlots1: '',
					nrofDownlinkSymbols1: '',
					nrofUplinkSymbols1: '',
					dlulTransmissionPeriodicity2: '',
					nrofDownlinkSlots2: '',
					nrofUplinkSlots2: '',
					nrofDownlinkSymbols2: '', 
					nrofUplinkSymbols2: '',
					prachRootSequenceIndex:'',
					prachRootSequenceValue:'', 
					plmn: [],
					periodicInformEnable: '',//Management Server
					periodicInformTime: '',
					periodicInformInterval: '',
					ntpEnable: '0',//ntp
					localTimeZone: '',
					ntpServer1: '',
					ntpServer2: '',
					ntpServer3: '',
					ntpServer4: '',
					ntpServer5: '',
					amf: [],//amf
					wan: [],
					lan: []
				},
				infoActiveName: ['gnb'],//------------------------------批量导入 详情+ 修改
				paramsImportEditDialogShow: false,
				paramsImportSaveDialogShow: false,
				gnbImportActiveName: ['gnb'],
				gnbImportSaveForm: {
					NCI: '',
					tac: '',
					ranac: '',
					plmnId:'',
					primary:'',
					sd: '0',
					sd_value:'',
					amfIp: '',
					default: '0',
					destinationNetwork: '',
					netmask: '',
					gateway: '',
					addressType: 'DHCP',
					ipAddress: '',
					subnetMask: '', //ipv4 static
					prefixLength: '',//ipv6 static
					bearType: 'NG',
					vlanName: '',
					vlanId: '',
					origin: '',
				},
				gnbImportSaveFormRules: {
					NCI:[{ required:true, validator: validateCommonRange, min: 0, max: 68719476735, errorMsg:'<%=rb.getString("FanWei")%>：0~68719476735,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					tac:[{ required:true, validator: validateCommonRange, min: 0, max: 16777215, errorMsg:'<%=rb.getString("FanWei")%>：0~16777215,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					ranac:[{ required:true, validator: validateCommonRange, min: 0, max: 255, errorMsg:'<%=rb.getString("FanWei")%>：0~255,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					plmnId:[{ required:true, validator: validatePlmnId, errorMsg:'<%=rb.getString("FanWei")%>：5~6 Digit,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					primary:[{ required:true, message:'<%=rb.getString("BiTian")%>'}],
					sd_value:[{ required:true, validator: validateCommonRange, min: 0, max: 16777215, errorMsg:'<%=rb.getString("FanWei")%>：0~16777215,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					destinationNetwork:[{required:true, validator: validateDestinationNetwork, trigger:'blur'}],	
					netmask:[{required:true, validator: validateNetmask, trigger:'blur'}],	
					gateway:[{required: true, validator: validateGateway, trigger:'blur'}],	
					ipAddress:[{required:true, validator: validateImportWanIpAddress, trigger:'blur'}],	
					subnetMask:[{required:true, validator: validateWanSubnetMask,trigger:'blur'}],	
					prefixLength:[{required:true, validator: validateWanPrefixLength,trigger:'blur'}],
					vlanName:[{ validator: validateWanVlanName, trigger:'blur'}],	
					vlanId:[{ validator: validateWanVlanID, trigger:'blur'}],
					amfIp:[{required:true, validator: validateAmfIpAddress, trigger:'blur'}],	
				},
				gnbImportAddOrEditForm: {
					gnbName: '',
					gnbLength: '',
					gnbId: '',
					pci: '',
					freqBandIndicator:'41',
					nrarfcndl:'515154',
					dlbandwidth:'100',
					ssbFrequency:'506910',
					nrarfcnul:'515154',
					ulbandwidth:'100',
					duplex_mode:'TDDMode',	//TDD  
					dlSubcarrierSpacing: '1',
					ulSubcarrierSpacing: '1',
					ulAntNum:'2',
					dlAntNum:'2',
					dlulTransmissionPeriodicity1:'6', //6
					nrofDownlinkSlots1: '7',
					nrofUplinkSlots1: '2',
					nrofDownlinkSymbols1: '4',
					nrofUplinkSymbols1: '4',
					dlulTransmissionPeriodicity2: '5',
					nrofDownlinkSlots2: '7',
					nrofUplinkSlots2: '2',
					nrofDownlinkSymbols2: '6', 
					nrofUplinkSymbols2: '4',
					prachRootSequenceIndex:'0',
					prachRootSequenceValue:'0', 
					plmn: [],
					periodicInformEnable: '0',//Management Server
					periodicInformTime: '',
					periodicInformInterval: '',
					ntpEnable: '0',//ntp
					localTimeZone: '',
					ntpServer1: '',
					ntpServer2: '',
					ntpServer3: '',
					ntpServer4: '',
					ntpServer5: '',
					amf: [],
					wan: [],
					lan: []
				},
				gnbImportAddOrEditRules: {
					gnbLength:[{required:true, validator: validateGnbLengthRange, min:22, max:32, errorMsg:'<%=rb.getString("FanWei")%>：22~32,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					gnbId:[{required:true, validator: validateImportGnbIdRange, min:0, max:4294967295, errorMsg:'<%=rb.getString("FanWei")%>0~4294967295,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					pci:[{required:true, validator: validateImportGnbIdRange, min:0, max:1007, errorMsg:'<%=rb.getString("FanWei")%>：0~1007,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					plmn: [{required:true, validator: validateImportPlmn, errorMsg:'<%=rb.getString("BiTian")%>' }],
					freqBandIndicator:[{ validator: validateNotRequireRange, min: 1, max: 1024, errorMsg:'<%=rb.getString("FanWei")%>：1~1024,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrarfcndl:[{ validator: validateNotRequireRange, min: 0, max: 3279165, errorMsg:'<%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					ssbFrequency:[{ validator: validateNotRequireRange, min: 0, max: 3279165, errorMsg:'<%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrarfcnul:[{ validator: validateNotRequireRange, min: 0, max: 3279165, errorMsg:'<%=rb.getString("FanWei")%>：0~3279165,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofDownlinkSlots1:[{ validator: validateNotRequireRange, min: 0, max: 320, errorMsg:'<%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofUplinkSlots1:[{ validator: validateNotRequireRange, min: 0, max: 320,  errorMsg:'<%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofDownlinkSymbols1:[{ validator: validateNotRequireRange, min: 0, max: 13,  errorMsg:'<%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofUplinkSymbols1:[{ validator: validateNotRequireRange, min: 0, max: 13, errorMsg:'<%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofDownlinkSlots2:[{ validator: validateNotRequireRange, min: 0, max: 320, errorMsg:'<%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofUplinkSlots2:[{ validator: validateNotRequireRange, min: 0, max: 320, errorMsg:'<%=rb.getString("FanWei")%>：0~320,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofDownlinkSymbols2:[{ validator: validateNotRequireRange, min: 0, max: 13, errorMsg:'<%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					nrofUplinkSymbols2:[{ validator: validateNotRequireRange, min: 0, max: 13, errorMsg:'<%=rb.getString("FanWei")%>：0~13,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					prachRootSequenceValue:[{ validator: validateImportPrachValueRange, trigger:'blur'}],
					periodicInformTime:[{ required:true, validator: commonImportPeriodTime, errorMsg:'requied', trigger:'blur'}],
					periodicInformInterval:[{required:true, validator: validateImportPeriodRange, min: 1, errorMsg:'<%=rb.getString("FanWei")%>：1~Infinite,<%=rb.getString("ZhengXing")%>', trigger:'blur'}],
					localTimeZone:[{ required:true, validator: commonImportPeriodTime, errorMsg:'requied', trigger:'blur'}],
					ntpServer1:[{ required:true, validator: commonImportPeriodTime, errorMsg:'requied', trigger:'blur'}],
				},
				oldParamsWanSelectRow:{},
				oldimportWanSelectRow:{},
            }
        },
        computed: {
			batchImportDialogTitle(){
				var vm = this;
				return vm.commonImportOperType == 'add' ? '<%=rb.getString("TianJia")%>' : '<%=rb.getString("XiuGai")%>'
			},
			gnbIpsecConfigAddDialogTitle(){
				var vm = this;
				return vm.advanceIpsecOperType == 'add' ? '<%=rb.getString("TianJia")%>' : '<%=rb.getString("XiuGai")%>'
			},
			addIPSecTunnelBtnShow(){
				var arr = this.gnbAddOrEditForm.ipsecTunnel;
				return arr.length<3? true : false;
			},
			groupShow() {
				var vm = this, type = 'BaiBNQ';
				return {
					AMFSetting: ['BaiBNQ'].includes(type),
					NETWORK: ['BaiBNQ'].includes(type),
					BTSSetting: ['BaiBNQ'].includes(type),
					SYSTEM: ['BaiBNQ'].includes(type)
				}
			},
			plmnIdList(){//AMF 模块需要 plmn 模块中的plmnId 值
				var vm = this, arr = [];
				this.gnbAddOrEditForm.plmn.filter((item)=>{
					item.plmnConfigList.filter((items)=>{
						var isExist = arr.some(itemss =>itemss == items.plmnId);
						if(!isExist && items.plmnId){
							arr.push(items.plmnId)
						} 
					})
				})
				return arr
			},
			importPlmnIdList(){//批量导入-修改 plmn id 下拉值
				var vm = this, arr = [];
				vm.gnbImportAddOrEditForm.plmn.filter((item)=>{ 
					item.plmnConfigList.filter((items)=>{
						var isExist = arr.some(itemss =>itemss == items.plmnId);
						if(!isExist && items.plmnId){
							arr.push(items.plmnId)
						} 
					})
				})
				return arr
			},
			vlanIdList(){
				var arr = [];
				this.gnbAddOrEditForm.wan.filter((item)=>{
					var isExist = arr.some(items =>items == item.vlanId);
					if(!isExist && item.vlanId){
						arr.push(item.vlanId)
					} 
				})
				return arr
			},
			importVlanIdList(){//import vlan id 下拉值
				var arr = [];
				this.gnbImportAddOrEditForm.wan.filter((item)=>{
					var isExist = arr.some(items =>items == item.vlanId);
					if(!isExist && item.vlanId){
						arr.push(item.vlanId)
					} 
				})
				return arr
			},
			vlanNameList(){
				var arr = [];
				this.gnbAddOrEditForm.wan.filter((item)=>{
					var isExist = arr.some(items =>items == item.vlanName);
					if(!isExist && item.vlanName){
						arr.push(item.vlanName)
					} 
				})
				return arr
			},
			importVlanNameList(){
				var arr = [];
				this.gnbImportAddOrEditForm.wan.filter((item)=>{
					var isExist = arr.some(items =>items == item.vlanName);
					if(!isExist && item.vlanName){
						arr.push(item.vlanName)
					} 
				})
				return arr
			}
        },
        watch: {
            'gnbAddOrEditForm.productType': function(val) {
                var vm = this,  params = { product_value: val };
                axios.post("${ctx}/plugandplay/policy/getTargetVersion.action", stringify(params)).then(function(response){ // 获取目标版本信息
					var data = response.data;
					vm.gnbTarVersionList = data ? data : [];
				});
                vm.gnbLicenseQuery.productType = val;//产品类型改变，license 查询时product 参数改变；
            },
            'gnbAddOrEditForm.upgradeEnable': function(val) { //升级开关关闭，不校验必填项，输入框可置灰
                var vm = this;
                if(val == '1'){
                	vm.$refs.gnbAddOrEditForm.validateField('originalVersion');
                	vm.$refs.gnbAddOrEditForm.validateField('targetVersion');
                }else{
                	vm.$refs.gnbAddOrEditForm.clearValidate('originalVersion');
                	vm.$refs.gnbAddOrEditForm.clearValidate('targetVersion');
                	vm.gnbAddVersionShow = false;
                }
            },
            'gnbAddOrEditForm.duplex_mode': function(val) {
                var vm = this;
                if(val == 'TDDMode'){
                	vm.gnbAddOrEditForm.dlSubcarrierSpacing = '1';
					vm.gnbAddOrEditForm.ulSubcarrierSpacing = '1';				
				}else if(val == 'FDDMode'){
                	vm.gnbAddOrEditForm.dlSubcarrierSpacing = '0';
					vm.gnbAddOrEditForm.ulSubcarrierSpacing = '0'
                }else{
					vm.gnbAddOrEditForm.dlSubcarrierSpacing = '';
					vm.gnbAddOrEditForm.ulSubcarrierSpacing = ''
				}
            },
    		gnbCurSelectVersionData(row){
    			if(row.length != 0){this.gnbVersionMessage = '';}
    		},
    		'specifyVersionType': function(val) {
                if(val == '0'){
                	this.$refs.gnbAddOrEditForm.validateField('originalVersion');
                }else{
                	this.gnbAddVersionShow = false;
                	this.gnbResultOriginalVersionList = []; //置空已配置的初始版本
                	this.$refs.gnbAddOrEditForm.clearValidate('originalVersion');
                }
            }
        },
        methods: {
			clearParamsValue(type){// cell more 清除默认值
				var vm = this,
					paramsKeyList = ['freqBandIndicator','nrarfcndl','dlbandwidth', 'ssbFrequency','nrarfcnul','ulbandwidth', 'duplex_mode','dlSubcarrierSpacing','ulSubcarrierSpacing', 'ulAntNum','dlAntNum','dlulTransmissionPeriodicity1', 'nrofDownlinkSlots1','nrofUplinkSlots1','nrofDownlinkSymbols1', 'nrofUplinkSymbols1','dlulTransmissionPeriodicity2','nrofDownlinkSlots2', 'nrofUplinkSlots2','nrofDownlinkSymbols2','nrofUplinkSymbols2', 'prachRootSequenceIndex','prachRootSequenceValue'];
				if(type == 'form'){
					paramsKeyList.map(function(prop){
						vm.gnbAddOrEditForm[prop]= '';						
					});
				}else{
					paramsKeyList.map(function(prop){
						vm.gnbImportAddOrEditForm[prop]= '';						
					});
				}
			},
			defaultParamsValue(type){// cell more 还原默认值
				vm = this,
					paramsKeyList = ['freqBandIndicator','nrarfcndl','dlbandwidth', 'ssbFrequency','nrarfcnul','ulbandwidth', 'duplex_mode','dlSubcarrierSpacing','ulSubcarrierSpacing', 'ulAntNum','dlAntNum','dlulTransmissionPeriodicity1', 'nrofDownlinkSlots1','nrofUplinkSlots1','nrofDownlinkSymbols1', 'nrofUplinkSymbols1','dlulTransmissionPeriodicity2','nrofDownlinkSlots2', 'nrofUplinkSlots2','nrofDownlinkSymbols2','nrofUplinkSymbols2', 'prachRootSequenceIndex','prachRootSequenceValue'],
					paramsValueList = ['41','515154','100', '506910','515154','100', 'TDDMode','1','1', '2','2','6', '7','2','4', '4','5','7', '2','6','4','0','0'];
				if(type == 'form'){
					paramsKeyList.map(function(prop){
						vm.gnbAddOrEditForm[prop]= paramsValueList[paramsKeyList.indexOf(prop)];						
					});
				}else{
					paramsKeyList.map(function(prop){
						vm.gnbImportAddOrEditForm[prop]= paramsValueList[paramsKeyList.indexOf(prop)];						
					});
				}
			},
			paramsImportTableEditClick(row, evt){//--------------------------------------------------------------- params config import
				var vm = this, commonParams = {};
				vm.commonImportRow = row;
				if(vm.gnbOperType == 'add'){
					commonParams.policyId = '${policyId}';
				}else{
					commonParams.policyId = vm.gnbPolicyId;
				}
				commonParams.serialNumber =  row.serial_number;  // 批量配置传递 sn, 公共配置传递default
				axios.post('${ctx}/gnb/pnp/getPnPPolicyConfigInfo.action', stringify(commonParams)).then(function(res){//查询所有参数接口
					var data = res.data;
					if(data){
						if(data.plmn && data.plmn.length > 0){
							data.plmn.map((item,index)=>{
								if(item.plmnConfigList && item.plmnConfigList.length > 0){
									item.plmnConfigList.map((items,idx)=>{
										items['plmnNrIndex'] = item.index;//向item.plmnConfigList中增加 plmnNrIndex 属性，取值为 item.index	
										items.index = items.index + '';
										if(items.sliceList && items.sliceList.length > 0){
											items.sliceList.map((itemss,idxs)=>{
												itemss.index = itemss.index + '';
											})
										}else{
											items['sliceList'] = [];
										}
									})
								}else{
									item['plmnConfigList'] = [];
								} 
							})
						}
						Object.assign(vm.gnbImportAddOrEditForm, data);							
					}					
				});
				vm.paramsImportEditDialogShow = true;
				vm.commonImportFileInfoShow = false;
			},
			paramsImportEditDialogSubmit(){//batch import: edit: 一级弹窗 提交
				var vm = this, params = {};
            	if(vm.gnbOperType == 'add'){
					params.policyId = '${policyId}';
				}else{
					params.policyId = vm.gnbPolicyId;
				}
				params.serialNumber = vm.commonImportRow.serial_number;
				params.gnbName = vm.gnbImportAddOrEditForm.gnbName;
                params.gnbLength = vm.gnbImportAddOrEditForm.gnbLength;
                params.gnbId = vm.gnbImportAddOrEditForm.gnbId;
				params.pci = vm.gnbImportAddOrEditForm.pci;
				params.freqBandIndicator = vm.gnbImportAddOrEditForm.freqBandIndicator;
				params.nrarfcndl = vm.gnbImportAddOrEditForm.nrarfcndl;
				params.dlbandwidth = vm.gnbImportAddOrEditForm.dlbandwidth;
				params.ssbFrequency = vm.gnbImportAddOrEditForm.ssbFrequency;
                params.nrarfcnul = vm.gnbImportAddOrEditForm.nrarfcnul;
				params.ulbandwidth = vm.gnbImportAddOrEditForm.ulbandwidth;
				params.duplex_mode = vm.gnbImportAddOrEditForm.duplex_mode;
				params.dlSubcarrierSpacing = vm.gnbImportAddOrEditForm.dlSubcarrierSpacing;
				params.ulSubcarrierSpacing = vm.gnbImportAddOrEditForm.ulSubcarrierSpacing;
				params.ulAntNum = vm.gnbImportAddOrEditForm.ulAntNum;
				params.dlAntNum = vm.gnbImportAddOrEditForm.dlAntNum;
				params.dlulTransmissionPeriodicity1 = vm.gnbImportAddOrEditForm.dlulTransmissionPeriodicity1;
				params.nrofDownlinkSlots1 = vm.gnbImportAddOrEditForm.nrofDownlinkSlots1;
				params.nrofUplinkSlots1 = vm.gnbImportAddOrEditForm.nrofUplinkSlots1;
				params.nrofDownlinkSymbols1 = vm.gnbImportAddOrEditForm.nrofDownlinkSymbols1;			
				params.nrofUplinkSymbols1 = vm.gnbImportAddOrEditForm.nrofUplinkSymbols1;
				params.dlulTransmissionPeriodicity2 = vm.gnbImportAddOrEditForm.dlulTransmissionPeriodicity2;
				params.nrofDownlinkSlots2 = vm.gnbImportAddOrEditForm.nrofDownlinkSlots2;
				params.nrofUplinkSlots2 = vm.gnbImportAddOrEditForm.nrofUplinkSlots2;
				params.nrofDownlinkSymbols2 = vm.gnbImportAddOrEditForm.nrofDownlinkSymbols2;
				params.nrofUplinkSymbols2 = vm.gnbImportAddOrEditForm.nrofUplinkSymbols2;
				params.prachRootSequenceIndex = vm.gnbImportAddOrEditForm.prachRootSequenceIndex;
                params.prachRootSequenceValue = vm.gnbImportAddOrEditForm.prachRootSequenceValue;
				params.plmn = JSON.stringify(vm.gnbImportAddOrEditForm.plmn);
				params.periodicInformEnable = vm.gnbImportAddOrEditForm.periodicInformEnable;
				params.periodicInformTime = vm.gnbImportAddOrEditForm.periodicInformTime;
				params.periodicInformInterval = vm.gnbImportAddOrEditForm.periodicInformInterval;
				params.ntpEnable = vm.gnbImportAddOrEditForm.ntpEnable;
				params.localTimeZone = vm.gnbImportAddOrEditForm.localTimeZone;
				params.ntpServer1 = vm.gnbImportAddOrEditForm.ntpServer1;
				params.ntpServer2 = vm.gnbImportAddOrEditForm.ntpServer2;
				params.ntpServer3 = vm.gnbImportAddOrEditForm.ntpServer3;
				params.ntpServer4 = vm.gnbImportAddOrEditForm.ntpServer4;
				params.ntpServer5 = vm.gnbImportAddOrEditForm.ntpServer5;
				params.amf = JSON.stringify(vm.gnbImportAddOrEditForm.amf);
				params.wan = JSON.stringify(vm.gnbImportAddOrEditForm.wan);
				params.lan = JSON.stringify(vm.gnbImportAddOrEditForm.lan);
				vm.$refs.gnbImportAddOrEditForm.validate(function(valid){
					if(valid){
						vm.gnbParamImportSaveBtnDisabled = true;
                        axios.post('${ctx}/gnb/pnp/addPnPPolicyConfigInfo.action',stringify(params)).then(function(res){
                        	var data = res.data;
                        	if(data){
                        		vm.gnbParamImportSaveBtnDisabled = false;
                        		if(data['success'] == true) {
                                    vm.$message({
    		    						message: '<%=rb.getString("ChengGong")%>',
    		    						type:'success',
    		    					});
                                    vm.$refs.paramsImportFileTable.refresh();
                                    vm.paramsImportEditDialogClose();
                                }else {
                                    vm.$message.error(data['message']);
                                }
                        	}
                        }).catch(function(){});
					}
				})
			},
			paramsImportEditDialogClose(){//batch import: edit: 一级弹窗 关闭
				var vm = this;
				vm.$refs.gnbImportAddOrEditForm.resetFields();
				vm.paramsImportEditDialogShow = false;
			},
			paramsImportSaveDialogSubmit(){//batch import: edit: 二级弹窗 提交
				var vm = this, params = {}, index = 'index';
				vm.$refs.gnbImportSaveForm.validate(function(valid){
					if(valid ){
						if(vm.commonImportOperModel == 'importPlmn'){
							params= {
								NCI: vm.gnbImportSaveForm.NCI,
								tac: vm.gnbImportSaveForm.tac,
								ranac: vm.gnbImportSaveForm.ranac,
								index: vm.gnbImportSaveForm.index ? vm.gnbImportSaveForm.index : '',
								plmnConfigList: []
							};
							if(vm.commonImportOperType == 'add'){
								if(vm.gnbImportAddOrEditForm.plmn.length == 0){
									params[index] = '1';
								}else{
									var idList=[];
									vm.gnbImportAddOrEditForm.plmn.map((item)=>{
										idList.push(item[index] + '');
									})
									params[index] = vm.createId(1,idList); 
								}
								vm.gnbImportAddOrEditForm.plmn.push(params);
							}else{
								var idx='';
								vm.gnbImportAddOrEditForm.plmn.map((item,indexs)=>{
									if(item[index] == params[index]){
										idx = indexs
									}
								})
								Object.assign(vm.gnbImportAddOrEditForm.plmn[idx],params)
							}
						}else if(vm.commonImportOperModel == 'importPlmnConfig'){//plmn setting
							params= {
								plmnNrIndex: vm.commonImportPlmnNrRows.index,//是 plmn nr 的index
								plmnId: vm.gnbImportSaveForm.plmnId,
								primary: vm.gnbImportSaveForm.primary,
								sliceList: []
							};
							if(vm.commonImportOperType == 'add'){						
								vm.gnbImportAddOrEditForm.plmn.map((item,index)=>{//将params 添加到 vm.gnbImportAddOrEditForm.plmn 当前父级对应  index ， plmnConfigList 数组中
									if(item.index == vm.commonImportPlmnNrRows.index){
										if(item.plmnConfigList.length == 0){
											params.index = '1';
										}else{
											var idList=[];
											item.plmnConfigList.map((items)=>{
												idList.push(items.index + '');
											})
											params.index = vm.createId(1,idList); 
										}
										item.plmnConfigList.push(params);
									}
								})
							}else{
								var idx='';	//将params 添加到 vm.gnbImportAddOrEditForm.plmn 当前父级对应  index ， plmnConfigList 数组中
								vm.gnbImportAddOrEditForm.plmn.map((item,index)=>{
									if(item.index == vm.commonImportPlmnNrRows.index){ //说明要祖级
										item.plmnConfigList.map((items,indexs)=>{
											if(items.plmnConfigIndex == vm.editImportPlmnCongigRows.plmnConfigIndex){
												idx = indexs
											}
										})
										Object.assign(item.plmnConfigList[idx],params)
									}
								})
							}
						}else if(vm.commonImportOperModel == 'importSliceConfig'){//sliceConfig 保存
							params= {
								plmnConfigIndex: vm.commonImportPlmnConfigRows.index,//是 plmnconfig的index
								sd: vm.gnbImportSaveForm.sd,
								sd_value: vm.gnbImportSaveForm.sd_value
							};
							if(vm.commonImportOperType == 'add'){
								vm.gnbImportAddOrEditForm.plmn.map((item,index)=>{//向 vm.gnbImportAddOrEditForm.plmn 中的 plmnConfigList 中的sliceList数组，添加数据
									if(item.index == vm.commonImportPlmnConfigRows.plmnNrIndex){
										item.plmnConfigList.map((items,indexs)=>{
											if(items.index == vm.commonImportPlmnConfigRows.index){
												if(items.sliceList.length == 0){
													params.index = '1';
												}else{
													var idList=[];
													items.sliceList.map((itemss)=>{
														idList.push(itemss.index + '');
													})
													params.index = vm.createId(1,idList); 
												}
												items.sliceList.push(params);
											}
										})
									}
								})
							}else{
								var idx='';//修改 vm.gnbImportAddOrEditForm.plmn 中的 plmnConfigList 中的sliceList数组中行数据
								vm.gnbImportAddOrEditForm.plmn.map((item,index)=>{
									if(item.index == vm.commonImportPlmnConfigRows.plmnNrIndex){
										item.plmnConfigList.map((items,indexs)=>{
											if(items.index == vm.commonImportPlmnConfigRows.index){
												items.sliceList.map((itemss,indexss)=>{
													if(itemss.index == vm.editImportSliceCongigRows.index){
														idx = indexss
													}
												})
												Object.assign(items.sliceList[idx],params)
											}
										})
									}
								})
							}
						}else if(vm.commonImportOperModel == 'importAMF') {
							var amfIdxStr = 'index';
							params= {
								amfIp: vm.gnbImportSaveForm.amfIp,
								plmnId: vm.gnbImportSaveForm.plmnId,
								default: vm.gnbImportSaveForm.default
							};
							if(vm.gnbImportAddOrEditForm.amf.length == 0){
								params[amfIdxStr] = '1';
							}else{
								var idList=[];
								vm.gnbImportAddOrEditForm.amf.map((item)=>{
									idList.push(item[amfIdxStr] + '');
								})
								params[amfIdxStr] = vm.createId(1,idList); 
							}
							vm.gnbImportAddOrEditForm.amf.push(params);
						}else if(vm.commonImportOperModel == 'importWan'){
							params= {
								origin: vm.gnbImportSaveForm.origin,
								addressType: vm.gnbImportSaveForm.addressType,
								ipAddress: vm.gnbImportSaveForm.ipAddress,
								subnetMask: vm.gnbImportSaveForm.subnetMask,
								prefixLength: vm.gnbImportSaveForm.prefixLength,
								gateway: vm.gnbImportSaveForm.gateway,
								bearType: vm.gnbImportSaveForm.bearType, 
								vlanName: vm.gnbImportSaveForm.vlanName,
								vlanId: vm.gnbImportSaveForm.vlanId,
								index: vm.gnbImportSaveForm.index ? vm.gnbImportSaveForm.index : '',
							};
							var isExist = false;//判断是否有相同的vlan name 和 vlan id
							if(vm.commonImportOperType == 'add'){
								if(vm.importAddVlanShow){
									vm.importVlanNameList.map((item) =>{
										if(vm.gnbImportSaveForm.vlanName && item == vm.gnbImportSaveForm.vlanName){
											isExist = true;
											vm.$message({
												type:'warning',
												message:'<%=rb.getString("YouXiangTongVLANMingCheng")%>'
											})
											return;
										}
									});
									vm.importVlanIdList.map((item) =>{
										if(vm.gnbImportSaveForm.vlanId && item == vm.gnbImportSaveForm.vlanId){
											isExist = true;
											vm.$message({
												type:'warning',
												message:'<%=rb.getString("YouXiangTongVLANId")%>'
											})
											return;
										}
									});
								}
								if(isExist)return;
							}else{
								if(vm.importAddVlanShow){
									vm.importVlanNameList.map((item) =>{
										if(item == vm.gnbImportSaveForm.vlanName && vm.gnbImportSaveForm.vlanName != vm.oldimportWanSelectRow.vlanName){
											isExist = true;
											vm.$message({
												type:'warning',
												message:'<%=rb.getString("YouXiangTongVLANMingCheng")%>'
											})
											return;
										}
									});
									vm.importVlanIdList.map((item) =>{
										if(item == vm.gnbImportSaveForm.vlanId && vm.gnbImportSaveForm.vlanId != vm.oldimportWanSelectRow.vlanId){
											isExist = true;
											vm.$message({
												type:'warning',
												message:'<%=rb.getString("YouXiangTongVLANId")%>'
											})
											return;
										}
									});
								}
								if(isExist)return;
							}
							if(vm.commonImportOperType == 'add'){
								if(vm.gnbImportAddOrEditForm.wan.length == 0){
									params[index] = '1';
								}else{
									var idList=[];
									vm.gnbImportAddOrEditForm.wan.map((item)=>{
										idList.push(item[index] + '');
									})
									params[index] = vm.createId(1,idList); 
								}
								vm.gnbImportAddOrEditForm.wan.push(params);
							}else{
								var idx='';
								vm.gnbImportAddOrEditForm.wan.map((item,indexs)=>{
									if(item[index] == params[index]){
										idx = indexs
									}
								})
								Object.assign(vm.gnbImportAddOrEditForm.wan[idx],params)
							}
						}else if(vm.commonImportOperModel == 'importLan'){
							params= {
								ipAddress: vm.gnbImportSaveForm.ipAddress,
								subnetMask: vm.gnbImportSaveForm.subnetMask,
								index: vm.gnbImportSaveForm.index ? vm.gnbImportSaveForm.index : '',
							};
							if(vm.commonImportOperType == 'add'){
								if(vm.gnbImportAddOrEditForm.lan.length == 0){
									params[index] = '1';
								}else{
									var idList=[];
									vm.gnbImportAddOrEditForm.lan.map((item)=>{
										idList.push(item[index] + '');
									})
									params[index] = vm.createId(1,idList); 
								}
								vm.gnbImportAddOrEditForm.lan.push(params);
							}else{
								var idx='';
								vm.gnbImportAddOrEditForm.lan.map((item,indexs)=>{
									if(item[index] == params[index]){
										idx = indexs
									}
								})
								Object.assign(vm.gnbImportAddOrEditForm.lan[idx],params)
							}							
						} 
						vm.paramsImportSaveDialogClose();
					}
				});
			},
			paramsImportSaveDialogClose (){//batch import: edit: 二级弹窗 关闭
				var vm = this;
				vm.$refs.gnbImportSaveForm.resetFields();
				vm.paramsImportSaveDialogShow = false;
			},
			paramsImportTableViewClick(row, evt){
				var vm = this, commonParams = {}, paramsKeyList = ['gnbName','gnbLength','gnbId','pci', 'freqBandIndicator','nrarfcndl','dlbandwidth', 'ssbFrequency','nrarfcnul','ulbandwidth','duplex_mode','dlSubcarrierSpacing','ulSubcarrierSpacing', 'ulAntNum','dlAntNum','dlulTransmissionPeriodicity1','nrofDownlinkSlots1','nrofUplinkSlots1','nrofDownlinkSymbols1','nrofUplinkSymbols1','dlulTransmissionPeriodicity2','nrofDownlinkSlots2', 'nrofUplinkSlots2','nrofDownlinkSymbols2','nrofUplinkSymbols2','prachRootSequenceIndex','prachRootSequenceValue','plmn','periodicInformEnable','periodicInformTime','periodicInformInterval','ntpEnable','localTimeZone','ntpServer1','ntpServer2','ntpServer3','ntpServer4','ntpServer5','amf','wan','lan'];
				vm.importFileInfoSN = row.serial_number;
				vm.importFileInfoCellName = row.gnb_name;
				if(vm.gnbOperType == 'add'){
					commonParams.policyId = '${policyId}';
				}else{
					commonParams.policyId = vm.gnbPolicyId;
				}
				commonParams.serialNumber = row.serial_number;
				vm.commonImportFileInfoShow = true;
				vm.commonImportFileShow = false;//params import
				vm.gnbAddVersionShow = false; //版本窗口
				vm.gnbLicenseImportShow = false; //license
				vm.gnbAdvanceTreeShow = false; // params advance tree
				vm.commonAddOrUpdateParamShow = false; //参数配置 commonAddOrUpdateParam
				vm.addIPSecTunnelDialogShow = false; // ipsec
				axios.post('${ctx}/gnb/pnp/getPnPPolicyConfigInfo.action', stringify(commonParams)).then(function(res){//查询所有参数接口
					var data = res.data;
					if(data){
						paramsKeyList.map(function(prop){
							vm.paramsImportInfo[prop]= data[prop];	
						});
					}					
				});
			},
			commonImportFileInfoCancel(){
				var vm = this;
				vm.commonImportFileInfoShow = false;
			},
			paramsImportTableDelClick(row, evt){
				var vm = this, confirmStr = '<%=rb.getString("QueRenShanChu")%>', params = {serialNumbers: row.serial_number};
				if(vm.gnbOperType == 'add'){
					params.policyId = '${policyId}';
				}else{
					params.policyId = vm.gnbPolicyId;
				}
				vm.commonImportFileInfoShow = false;
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/gnb/pnp/delPnPPolicyConfigInfo.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.paramsImportFileTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {})
				vm.commonImportFileShow = false;//params import
				vm.commonImportFileInfoShow = false; // params import info
				vm.gnbAddVersionShow = false; //版本窗口
				vm.gnbLicenseImportShow = false; //license
				vm.gnbAdvanceTreeShow = false; // params advance tree
				vm.commonAddOrUpdateParamShow = false; //参数配置 commonAddOrUpdateParam
				vm.addIPSecTunnelDialogShow = false; // ipsec
			},
			gnbParamImportFileClick(){
				var vm = this;
				vm.commonImportFileLoading = false; // 页面loading
				vm.commonImportFileDisabled = false; //提交按钮置灰
				vm.commonImportFileShow = true;//params import
				vm.commonImportFileInfoShow = false; // params import info
				vm.gnbAddVersionShow = false; //版本窗口
				vm.gnbLicenseImportShow = false; //license
				vm.gnbAdvanceTreeShow = false; // params advance tree
				vm.commonAddOrUpdateParamShow = false; //参数配置 commonAddOrUpdateParam
				vm.addIPSecTunnelDialogShow = false; // ipsec
				vm.$refs.gnbImportRuleForm.resetFields();
			},
			gnbCheckFile(res,file){   //文件上传成功函数 发送请求，校验device文件内容 
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
					vm.$refs.paramsImportFileTable.refresh();
					vm.gnbCloseFileSelect();
					vm.commonImportFileShow = false;
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				var fileList = vm.$refs.upload.uploadFiles;//修改已选择文件状态  
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
			gnbFileChange(file,fileList){ //选择文件后，校验格式，并赋值页面显示  
				var vm = this;
				vm.fileName = file.name;
				vm.fileParams.FileName = file.name;
			},
			gnbFileSelect(){  // 选择文件
				var vm =this;
				vm.$refs.upload.clearFiles();
				vm.$refs['file_up'].click();
			},
			gnbCloseFileSelect(){// 移除导入文件
				var vm = this;
				vm.fileName = '';			
				vm.$refs.upload.clearFiles();
			},
			gnbBeforeUpload(file){//文件上传之前 
				var vm = this, importUrl, fileName = file.name,fileSize = file.size, commonId = '', fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};
					if(vm.gnbOperType == 'add'){
						commonId = '${policyId}';
					}else{
						commonId = vm.gnbPolicyId;
					}
				fd.append('operType',vm.gnbImportRuleForm.operType);//导入类型
				fd.append('policyId', commonId);//文件大小
				fd.append('uploadFile',file); //文件流
				vm.commonImportFileLoading = true;
				axios.post('${ctx}/gnb/pnp/importBatchConfigInfos.action',fd,config).then(function(res){
					var data = res.data;
					if(data["success"]){	
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.$refs.paramsImportFileTable.refresh();
						vm.gnbParamsImportCancel();
					}else{
						vm.gnbParamsImportCancel();
						if(data["msg"] == "1"){
							vm.$message.error('<%=rb.getString("DaoRuShiBai")%>');
						}else if(data["msg"] == "2"){
							vm.$refs.paramsImportFileTable.refresh();
							vm.downloadVisible = true;
					   }else{
						   vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
					   }
					}
					vm.commonImportFileLoading = false; // 页面loading
					vm.commonImportFileDisabled = false; //提交按钮置灰
				})
				return false;
			},
			closeDownloadDialog(){
				var vm = this;
				vm.downloadVisible = false;
			},
			downloadErrorFile(){
				exportByForm('${ctx}/gnb/pnp/downloadFailureFile.action',{});
				this.downloadVisible = false;
			},		
			gnbParamsImportSubmit(){//coordinate 导入提交
				var vm = this;
				vm.$refs.gnbImportRuleForm.validate((valid) => {
					if(valid){
						vm.$refs.upload.submit();
						vm.commonImportFileLoading = true; // 页面loading
						vm.commonImportFileDisabled = true; //提交按钮置灰
					}
				})
			},
			gnbParamsImportCancel(){//导入取消
				var vm = this;
				vm.fileList = [];
				vm.fileName = '';
				vm.importRuleForm = {
					operType: "0",
					filePath: ""
				}
				vm.$refs.gnbImportRuleForm.resetFields();
				vm.commonImportFileShow = false;
			},
			gnbExportTemplate(){//下载模板
				exportByForm('${ctx}/gnb/pnp/downloadSelfConfigTemp.action', {});
			},
			gnbParamExportClick(){//导出
				/*var vm = this, params = {search_text : this.importParamsQuery.searchText, timeZone : timeZone}
				if(vm.gnbOperType == 'modify'){
					params.policyId = vm.curPolicyId;
				}
				exportByForm('',params);*/
			},
			importParamClick(row, operType, operModel, propsRow){//新建，修改按钮点击：行数据，操作类型(add,edit)，操作模块(amf,wan,lan....), plmn nr  行数据
				var vm = this;
				vm.commonImportOperType = operType;
				vm.commonImportOperModel = operModel;
				if(operModel == 'importWan'){
					vm.importAddVlanShow = false;
					if(operType == 'edit'){
						Object.assign(vm.gnbImportSaveForm, row) 
						vm.oldimportWanSelectRow = row;
					}else{
						vm.gnbImportSaveForm.addressType = 'DHCP';
						vm.gnbImportSaveForm.bearType =  'NG';
						vm.gnbImportSaveForm.ipAddress = '';
						vm.gnbImportSaveForm.subnetMask = '';
						vm.gnbImportSaveForm.prefixLength = '';
						vm.gnbImportSaveForm.vlanName = '';
						vm.gnbImportSaveForm.vlanId = '';
					}
				}else if(operModel == 'importLan'){
					vm.gnbImportSaveForm.addressType = 'Static';
					if(operType == 'edit'){
						Object.assign(vm.gnbImportSaveForm, row) 
					}else{
						vm.gnbImportSaveForm.ipAddress = '';
						vm.gnbImportSaveForm.subnetMask = '';
					}
				}else if(operModel == 'importPlmn'){
					if(operType == 'edit'){
						Object.assign(vm.gnbImportSaveForm, row) 
					}
				}else if(operModel == 'importPlmnConfig'){
					vm.commonImportPlmnNrRows = propsRow;
					if(operType == 'edit'){
						vm.editImportPlmnCongigRows = row;
						Object.assign(vm.gnbImportSaveForm, row) 
					}
				}else if(operModel == 'importSliceConfig'){
					vm.commonImportPlmnConfigRows = propsRow;
					if(operType == 'edit'){
						vm.editImportSliceCongigRows = row;
						Object.assign(vm.gnbImportSaveForm, row) 
					}else{
						vm.gnbImportSaveForm.sd = '0';
					}
				}
				vm.paramsImportSaveDialogShow = true; //参数配置 commonAddOrUpdateParam
			},
			importParamDelClick(row, operModel, propsRow){
				var vm = this, confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					if(operModel == 'importPlmn'){//plmn
						vm.gnbImportAddOrEditForm.plmn.map(function(item,index){
							if(item.index == row.index){
								vm.gnbImportAddOrEditForm.plmn = vm.gnbImportAddOrEditForm.plmn.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}else if( operModel == 'importPlmnConfig'){//plmnConfig setting
						vm.gnbImportAddOrEditForm.plmn.map(function(item,index){
							if(item.index == propsRow.index){ // plmn nr
								item.plmnConfigList = item.plmnConfigList.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}else if ( operModel == 'importSliceConfig'){//删除当前操作 sliceConfig 的行数据
						vm.gnbImportAddOrEditForm.plmn.map(function(item,index){
							if(item.index == propsRow.plmnNrIndex){
								item.plmnConfigList.map(function(items,indexs){ //propsRow: plmn config
									if(items.index == propsRow.index){
										items.sliceList = items.sliceList.filter((itemss)=>{
											return itemss.index != row.index
										})
									}
								})
							}
						})
					}else if(operModel == 'importAMF'){//advance / amf - AMF IP List: delete
						vm.gnbImportAddOrEditForm.amf.map(function(item,index){
							if(item.index == row.index){
								vm.gnbImportAddOrEditForm.amf = vm.gnbImportAddOrEditForm.amf.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}else if(operModel == 'importWan'){//advance / Static Routing List: delete
						vm.gnbImportAddOrEditForm.wan.map(function(item,index){
							if(item.index == row.index){
								vm.gnbImportAddOrEditForm.wan = vm.gnbImportAddOrEditForm.wan.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}else if(operModel == 'importLan'){
						vm.gnbImportAddOrEditForm.lan.map(function(item,index){
							if(item.index == row.index){
								vm.gnbImportAddOrEditForm.lan = vm.gnbImportAddOrEditForm.lan.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}
				})
			},
			commonAddOrUpdateParamClick(row, operType, operModel, propsRow){//新建，修改按钮点击：行数据，操作类型，操作模块, plmn nr  行数据    ------------------------------- advance 
				var vm = this;
				vm.commonAdvanceOperType = operType;
				vm.commonAdvanceOperModel = operModel;
				if(operType == 'add'){
					vm.commonAddOrUpdateParamTitle = '<%=rb.getString("TianJia")%>';
					vm.$refs.commonAddEditForm.resetFields();
				}else{
					vm.commonAddOrUpdateParamTitle = '<%=rb.getString("XiuGai")%>';
				}
				if(operModel == 'router'){// Static Routing List
					if(operType == 'edit'){
						Object.assign(vm.commonAddEditForm, row) 
					}
				}
				if(operModel == 'wan'){
					vm.addVlanShow = false;
					if(operType == 'edit'){
						Object.assign(vm.commonAddEditForm, row);
						vm.oldParamsWanSelectRow = row;
					}else{
						vm.commonAddEditForm.addressType = 'DHCP';
						vm.commonAddEditForm.bearType =  'NG';
						vm.commonAddEditForm.ipAddress = '';
						vm.commonAddEditForm.subnetMask = '';
						vm.commonAddEditForm.prefixLength = '';
					}
				}
				if(operModel == 'lan'){
					vm.commonAddEditForm.addressType = 'Static';
					if(operType == 'edit'){
						Object.assign(vm.commonAddEditForm, row) 
					}else{
						vm.commonAddEditForm.ipAddress = '';
						vm.commonAddEditForm.subnetMask = '';
					}
				}
				if(operModel == 'plmn'){//plmn NR
					if(operType == 'edit'){
						Object.assign(vm.commonAddEditForm, row) 
					}
				}
				if(operModel == 'plmnConfig'){//plmn setting
					vm.commonPlmnNrRows = propsRow;
					if(operType == 'edit'){
						vm.editPlmnCongigRows = row;
						Object.assign(vm.commonAddEditForm, row) 
					}
				}
				if(operModel == 'sliceConfig'){
					vm.commonPlmnConfigRows = propsRow;
					if(operType == 'edit'){
						vm.editSliceCongigRows = row;
						Object.assign(vm.commonAddEditForm, row) 
					}else{
						vm.commonAddEditForm.sd = '0';
					}
				}
				vm.commonAddOrUpdateParamShow = true; //参数配置 commonAddOrUpdateParam
				vm.gnbAddVersionShow = false; //版本窗口
				vm.gnbLicenseImportShow = false; //license
				vm.commonImportFileShow = false; //params import
				vm.commonImportFileInfoShow = false; // params import info
				vm.gnbAdvanceTreeShow = false; // params advance tree
				vm.addIPSecTunnelDialogShow = false; // ipsec
			},
			commonAddOrUpdateParamSubmit(){//advance: 新建，修改提交
				var vm = this, params = {}, index = 'index';
				vm.$refs.commonAddEditForm.validate(function(valid){
					if(valid ){
						if(vm.commonAdvanceOperModel == 'plmn'){
							params= {
								NCI: vm.commonAddEditForm.NCI,
								tac: vm.commonAddEditForm.tac,
								ranac: vm.commonAddEditForm.ranac,
								plmnConfigList: [],
								index: vm.commonAddEditForm.index ? vm.commonAddEditForm.index : ''
							};
							if(vm.commonAdvanceOperType == 'add'){
								if(vm.gnbAddOrEditForm.plmn.length == 0){
									params[index] = '1';
								}else{
									var idList=[];
									vm.gnbAddOrEditForm.plmn.map((item)=>{
										idList.push(item[index] + '');
									})
									params[index] = vm.createId(1,idList); 
								}
								vm.gnbAddOrEditForm.plmn.push(params);
							}else{
								var idx='';
								vm.gnbAddOrEditForm.plmn.map((item,indexs)=>{
									if(item[index] == params[index]){
										idx = indexs
									}
								})
								Object.assign(vm.gnbAddOrEditForm.plmn[idx],params)
							}
						}
						if(vm.commonAdvanceOperModel == 'plmnConfig'){//plmn setting
							params= {
								plmnNrIndex: vm.commonPlmnNrRows.index,//是 plmn nr 的index
								plmnId: vm.commonAddEditForm.plmnId,
								primary: vm.commonAddEditForm.primary,
								sliceList:[]
							};
							if(vm.commonAdvanceOperType == 'add'){//将params 添加到 vm.gnbAddOrEditForm.plmn 当前父级对应  index ， plmnConfigList 数组中
								vm.gnbAddOrEditForm.plmn.map((item,index)=>{
									if(item.index == vm.commonPlmnNrRows.index){
										if(item.plmnConfigList.length == 0){
											params.index = '1';
										}else{
											var idList=[];
											item.plmnConfigList.map((items)=>{
												idList.push(items.index + '');
											})
											params.index = vm.createId(1,idList); 
										}
										item.plmnConfigList.push(params);
									}
								})
							}else{
								var idx='';	 //将params 添加到 vm.gnbAddOrEditForm.plmn 当前父级对应  index ， plmnConfigList 数组中
								vm.gnbAddOrEditForm.plmn.map((item,index)=>{
									if(item.index == vm.commonPlmnNrRows.index){ //说明要祖级
										item.plmnConfigList.map((items,indexs)=>{
											if(items.plmnConfigIndex == vm.editPlmnCongigRows.plmnConfigIndex){
												idx = indexs
											}
										})
										Object.assign(item.plmnConfigList[idx],params)
									}
								})
							}
							
						}
						if(vm.commonAdvanceOperModel == 'sliceConfig'){//sliceConfig 保存
							params= {
								plmnConfigIndex: vm.commonPlmnConfigRows.index,//是 plmnconfig的index
								sd: vm.commonAddEditForm.sd,
								sd_value: vm.commonAddEditForm.sd_value
							};
							if(vm.commonAdvanceOperType == 'add'){
								vm.gnbAddOrEditForm.plmn.map((item,index)=>{
									if(item.index == vm.commonPlmnConfigRows.plmnNrIndex){ //plmn nr 中的 index
										item.plmnConfigList.map((items,indexs)=>{
											if(items.index == vm.commonPlmnConfigRows.index){
												if(items.sliceList.length == 0){
													params.index = '1';
												}else{
													var idList=[];
													items.sliceList.map((itemss)=>{
														idList.push(itemss.index + '');
													})
													params.index = vm.createId(1,idList); 
												}
												items.sliceList.push(params);
											}
										})
									}
								})
							}else{
								var idx=''; //修改 vm.gnbAddOrEditForm.plmn 中的 plmnConfigList 中的sliceList数组中行数据
								vm.gnbAddOrEditForm.plmn.map((item,index)=>{
									if(item.index == vm.commonPlmnConfigRows.plmnNrIndex){
										item.plmnConfigList.map((items,indexs)=>{
											if(items.index == vm.commonPlmnConfigRows.index){
												items.sliceList.map((itemss,indexss)=>{
													if(itemss.index == vm.editSliceCongigRows.index){
														idx = indexss
													}
												})
												Object.assign(items.sliceList[idx],params)
											}
										})
									}
								})
							}
						}
						if(vm.commonAdvanceOperModel == 'AMF') {
							var amfIdxStr = 'index';
							params= {
								amfIp: vm.commonAddEditForm.amfIp,
								plmnId: vm.commonAddEditForm.plmnId,
								default: vm.commonAddEditForm.default
							};
							if(vm.gnbAddOrEditForm.amf.length == 0){
								params[amfIdxStr] = '1';

							}else{
								var idList=[];
								vm.gnbAddOrEditForm.amf.map((item)=>{
									idList.push(item[amfIdxStr] + '');
								})
								params[amfIdxStr] = vm.createId(1,idList); 
							}
							vm.gnbAddOrEditForm.amf.push(params);
						}
						if(vm.commonAdvanceOperModel == 'router'){
							params= {
								destinationNetwork: vm.commonAddEditForm.destinationNetwork,
								netmask: vm.commonAddEditForm.netmask,
								gateway: vm.commonAddEditForm.gateway,
								index: vm.commonAddEditForm.index ? vm.commonAddEditForm.index : ''
							};
							if(vm.commonAdvanceOperType == 'add'){
								if(vm.gnbAddOrEditForm.staticRouting.length == 0){
									params[index] = '1';
								}else{
									var idList=[];
									vm.gnbAddOrEditForm.staticRouting.map((item)=>{
										idList.push(item[index] + '');
									})
									params[index] = vm.createId(1,idList); 
								}
								vm.gnbAddOrEditForm.staticRouting.push(params);
							}else{
								var idx='';
								vm.gnbAddOrEditForm.staticRouting.map((item,indexs)=>{
									if(item[index] == params[index]){
										idx = indexs
									}
								})
								Object.assign(vm.gnbAddOrEditForm.staticRouting[idx],params)
							}
						}
						if(vm.commonAdvanceOperModel == 'wan'){
							params= {
								origin: vm.commonAddEditForm.origin,
								addressType: vm.commonAddEditForm.addressType,
								ipAddress: vm.commonAddEditForm.ipAddress,
								subnetMask: vm.commonAddEditForm.subnetMask,
								prefixLength: vm.commonAddEditForm.prefixLength,
								gateway: vm.commonAddEditForm.gateway,
								bearType: vm.commonAddEditForm.bearType, 
								vlanName: vm.commonAddEditForm.vlanName,
								vlanId: vm.commonAddEditForm.vlanId,
								index: vm.commonAddEditForm.index ? vm.commonAddEditForm.index : ''
							};
							var isExist = false;//判断是否有相同的vlan name 和 vlan id
							if(vm.commonAdvanceOperType == 'add'){
								if(vm.addVlanShow){
									vm.vlanNameList.map((item) =>{
										if(vm.commonAddEditForm.vlanName && item == vm.commonAddEditForm.vlanName){
											isExist = true;
											vm.$message({
												type:'warning',
												message:'<%=rb.getString("YouXiangTongVLANMingCheng")%>'
											})
											return;
										}
									});
									vm.vlanIdList.map((item) =>{
										if(vm.commonAddEditForm.vlanId && item == vm.commonAddEditForm.vlanId){
											isExist = true;
											vm.$message({
												type:'warning',
												message:'<%=rb.getString("YouXiangTongVLANId")%>'
											})
											return;
										}
									});
								}
								if(isExist)return;
							}else{
								if(vm.addVlanShow){
									vm.vlanNameList.map((item) =>{
										if(item == vm.commonAddEditForm.vlanName && vm.commonAddEditForm.vlanName != vm.oldParamsWanSelectRow.vlanName){
											isExist = true;
											vm.$message({
												type:'warning',
												message:'<%=rb.getString("YouXiangTongVLANMingCheng")%>'
											})
											return;
										}
									});
									vm.vlanIdList.map((item) =>{
										if(item == vm.commonAddEditForm.vlanId && vm.commonAddEditForm.vlanId != vm.oldParamsWanSelectRow.vlanId){
											isExist = true;
											vm.$message({
												type:'warning',
												message:'<%=rb.getString("YouXiangTongVLANId")%>'
											})
											return;
										}
									});
								}
								if(isExist)return;
							}
							if(vm.commonAdvanceOperType == 'add'){
								if(vm.gnbAddOrEditForm.wan.length == 0){
									params[index] = '1';
								}else{
									var idList=[];
									vm.gnbAddOrEditForm.wan.map((item)=>{
										idList.push(item[index] + '');
									})
									params[index] = vm.createId(1,idList); 
								}
								vm.gnbAddOrEditForm.wan.push(params);
							}else{
								var idx='';
								vm.gnbAddOrEditForm.wan.map((item,indexs)=>{
									if(item[index] == params[index]){
										idx = indexs
									}
								})
								Object.assign(vm.gnbAddOrEditForm.wan[idx],params)
							}
						}
						if(vm.commonAdvanceOperModel == 'lan'){
							params= {
								ipAddress: vm.commonAddEditForm.ipAddress,
								subnetMask: vm.commonAddEditForm.subnetMask,
								index: vm.commonAddEditForm.index ? vm.commonAddEditForm.index : ''
							};
							if(vm.commonAdvanceOperType == 'add'){
								if(vm.gnbAddOrEditForm.lan.length == 0){
									params[index] = '1';
								}else{
									var idList=[];
									vm.gnbAddOrEditForm.lan.map((item)=>{
										
										idList.push(item[index] + '');
									})
									params[index] = vm.createId(1,idList); 
								}
								vm.gnbAddOrEditForm.lan.push(params);
							}else{
								//update 
								var idx='';
								vm.gnbAddOrEditForm.lan.map((item,indexs)=>{
									if(item[index] == params[index]){
										idx = indexs
									}
								})
								Object.assign(vm.gnbAddOrEditForm.lan[idx],params)
							}							
						} 
						vm.commonAddOrUpdateParamCancel();
					}
				});
			},
			commonDelParamClick(row, operModel, propsRow){//advance: delete
				var vm = this, confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					if(operModel == 'plmn'){//plmn
						vm.gnbAddOrEditForm.plmn.map(function(item,index){
							if(item.index == row.index){
								vm.gnbAddOrEditForm.plmn = vm.gnbAddOrEditForm.plmn.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}else if( operModel == 'plmnConfig'){//plmnConfig setting
						vm.gnbAddOrEditForm.plmn.map(function(item,index){
							if(item.index == propsRow.index){ // plmn nr
								item.plmnConfigList = item.plmnConfigList.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}else if ( operModel == 'sliceConfig'){//删除当前操作 sliceConfig 的行数据
						vm.gnbAddOrEditForm.plmn.map(function(item,index){
							if(item.index == propsRow.plmnNrIndex){
								item.plmnConfigList.map(function(items,indexs){ //propsRow: plmn config
									if(items.index == propsRow.index){
										items.sliceList = items.sliceList.filter((itemss)=>{
											return itemss.index != row.index
										})
									}
								})
							}
						})
					}else if(operModel == 'AMF'){//advance / amf - AMF IP List: delete
						vm.gnbAddOrEditForm.amf.map(function(item,index){
							if(item.index == row.index){
								vm.gnbAddOrEditForm.amf = vm.gnbAddOrEditForm.amf.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}else if(operModel == 'router'){//advance / Static Routing List: delete
						vm.gnbAddOrEditForm.staticRouting.map(function(item,index){
							if(item.index == row.index){
								vm.gnbAddOrEditForm.staticRouting = vm.gnbAddOrEditForm.staticRouting.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}else if(operModel == 'wan'){
						vm.gnbAddOrEditForm.wan.map(function(item,index){
							if(item.index == row.index){
								vm.gnbAddOrEditForm.wan = vm.gnbAddOrEditForm.wan.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}else if(operModel == 'lan'){
						vm.gnbAddOrEditForm.lan.map(function(item,index){
							if(item.index == row.index){
								vm.gnbAddOrEditForm.lan = vm.gnbAddOrEditForm.lan.filter((items)=>{
									return items.index != row.index
								})
							}
						})
					}
				})
			},
			commonAddOrUpdateParamCancel(){//advance: 关闭右侧窗口
				var vm = this;
				vm.commonAddOrUpdateParamShow = false;
			},
			addVlanClick(type){// 新增VLAN 按钮点击
				var vm = this;
				if(type == 'params'){
					vm.commonAddEditForm.vlanId = '';
					vm.addVlanShow = true;
				}else{
					vm.gnbImportSaveForm.vlanId = '';
					vm.importAddVlanShow = true;
				}
			},
			closeAddVlanClick(type){// 取消 新增VLAN 按钮点击
				var vm = this;
				if(type == 'params'){
					vm.addVlanShow = false;
					vm.commonAddEditForm.vlanId = '';
					vm.commonAddEditForm.vlanName = '';
				}else{
					vm.importAddVlanShow = false;
					vm.gnbImportSaveForm.vlanId = '';
					vm.gnbImportSaveForm.vlanName = '';
				}
			},
			addIPSecTunnelDialogOpen(row,operType){// 打开新增 IPSecTunnel 弹窗 行数 操作类型（add,edit,del） 操作模块(ipsec)
				var vm = this, idxStr = 'ipsec_index';
				vm.advanceIpsecOperType = operType;//操作类型 add edit
				if(operType == 'edit'){
					Object.keys(vm.addIPSecDialogForm).map((key)=>{
						var value = row[key], endVal = '', startVal = '';
						if(key != 'key_lefe_type' && key != 'ike_lefe_time_type' && key != 'rekey_margin_type' && key != 'dpddelay_type'){
							if(key == 'key_lefe'){
								if(value){
									endVal = value.slice(-1);
									startVal = value.slice(0,-1);
									vm.addIPSecDialogForm.key_lefe = startVal;
									vm.addIPSecDialogForm.key_lefe_type = endVal;
								}
							}else if(key == 'ike_lefe_time'){
								if(value){
									endVal = value.slice(-1);
									startVal = value.slice(0,-1);
									vm.addIPSecDialogForm.ike_lefe_time = startVal;
									vm.addIPSecDialogForm.ike_lefe_time_type = endVal;
								}
							}else if(key == 'rekey_margin'){
								if(value){
									endVal = value.slice(-1);
									startVal = value.slice(0,-1);
									vm.addIPSecDialogForm.rekey_margin = startVal;
									vm.addIPSecDialogForm.rekey_margin_type = endVal;
								}
							}else if(key == 'dpddelay'){
								if(value){
									endVal = value.slice(-1);
									startVal = value.slice(0,-1);
									vm.addIPSecDialogForm.dpddelay = startVal;
									vm.addIPSecDialogForm.dpddelay_type = endVal;
								}
							}else{
								vm.addIPSecDialogForm[key] = row[key] ? row[key] : '';
							}
						}
					})
					vm.addIPSecDialogForm[idxStr] = row[idxStr];
				}
				vm.addIPSecTunnelDialogShow = true; //ipsec
				vm.gnbAddVersionShow = false; //版本窗口
				vm.gnbLicenseImportShow = false; //license
				vm.commonImportFileShow = false; //params import
				vm.commonImportFileInfoShow = false; // params import info
				vm.gnbAdvanceTreeShow = false; // params advance tree
				vm.commonAddOrUpdateParamShow = false; //参数配置 commonAddOrUpdateParam
			},
			addIPSecTunnelDialogSubmit(){// 新增 IPSecTunnel 提交
				var vm = this, params = {}, codes = { 'IPSec':'ipsecTunnel'}, idxStr = 'ipsec_index', optTb = codes['IPSec'];
				Object.keys(vm.addIPSecDialogForm).forEach(function(key){
					if(key != 'key_lefe_type' && key != 'ike_lefe_time_type' && key != 'rekey_margin_type' && key != 'dpddelay_type'){
						if(key == 'key_lefe'){
							var value = vm.addIPSecDialogForm[key];
							if(value){
								params[key] = vm.addIPSecDialogForm.key_lefe + '' + vm.addIPSecDialogForm.key_lefe_type
							}
						}else if(key == 'ike_lefe_time'){
							var value = vm.addIPSecDialogForm[key];
							if(value){
								params[key] = vm.addIPSecDialogForm.ike_lefe_time + '' + vm.addIPSecDialogForm.ike_lefe_time_type
							}
						}else if(key == 'rekey_margin'){
							var value = vm.addIPSecDialogForm[key];
							if(value){
								params[key] = vm.addIPSecDialogForm.rekey_margin + '' + vm.addIPSecDialogForm.rekey_margin_type
							}
						}else if(key == 'dpddelay'){
							var value = vm.addIPSecDialogForm[key];
							if(value){
								params[key] = vm.addIPSecDialogForm.dpddelay + '' + vm.addIPSecDialogForm.dpddelay_type
							}
						}else{
							params[key] = vm.addIPSecDialogForm[key];
						}
					}
				})
				vm.$refs.addIPSecDialogForm.validate(function(valid){
					if(valid){
						var isExist = false;
						if(vm.advanceIpsecOperType == 'add'){
							isExist = vm.gnbAddOrEditForm[optTb].some(item =>item.gateway == vm.addIPSecDialogForm.gateway);
						}else{
							vm.gnbAddOrEditForm[optTb].map((item)=>{
								if(item.gateway == vm.addIPSecDialogForm.gateway){
									if(item[idxStr] != vm.addIPSecDialogForm[idxStr]){
										isExist = true;
									}
								}
							})
						}
						if(isExist){
							vm.$message.warning('Gateway already exists');
							return;
						}
						if(vm.advanceIpsecOperType == 'add'){
							if(vm.gnbAddOrEditForm[optTb].length == 0){
								params[idxStr] = '1'
							}else{
								var idList=[];
								vm.gnbAddOrEditForm[optTb].map((item)=>{
									idList.push(item[idxStr] + ''); // + ''
								})
								params[idxStr] = vm.createId(1,idList); 
							}
							vm.gnbAddOrEditForm[optTb].push(params);
						}else{
							var idx='';
							vm.gnbAddOrEditForm[optTb].map((item,index)=>{
								if(item[idxStr] == params[idxStr]){
									idx = index
								}
							})
							Object.assign(vm.gnbAddOrEditForm[optTb][idx],params)
						}
						vm.addIPSecTunnelDialogShow = false;
					}
				})
			},
			delIPSecTunnelList(row,operModel){// 删除 IPSec Tunnel
				var vm = this, codes = {'IPSec':'ipsecTunnel'}, idxStr = 'ipsec_index', optTb = codes[operModel];
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					vm.gnbAddOrEditForm[optTb].map(function(item,index){
						if(item[idxStr] == row[idxStr]){
							vm.gnbAddOrEditForm[optTb] = vm.gnbAddOrEditForm[optTb].filter((items)=>{
								return items[idxStr] != row[idxStr]
							})
						}
					})
					vm.$nextTick(function(){
						['NGAP_InterfaceBinding','NGU_InterfaceBinding'].map((key)=>{
							if(vm.gnbAddOrEditForm[key] == row.tunnel_name){
								if(vm.tunnelNameList.length > 0){
									vm.gnbAddOrEditForm[key] = vm.tunnelNameList[0]
								}else{
									vm.gnbAddOrEditForm[key] = ''
								}
							}
						})
					})
				})
			},
			closeAddIPSecTunnelDialog(){// 关闭 IPSecTunnel弹窗
				var vm = this,
					params = {
						tunnel_enable:'0',
						tunnel_name:'',
						left_auth:'psk',
						right_auth:'psk',
						gateway:'10.10.10.10',
						right_subnet:'0.0.0.0/0',
						right_id:'C=CH,O=strongSwan,CN=server',
						secret_key:'clientKey.der',
						tunnel_left:'%defaultroute',
						right_secretkey: '',
						left_id:'C=CH,O=strongSwan,CN=server',
						left_cert:'',
						left_source_ip:'%config',
						left_subnet:'',
						fragmentation:'yes',
						ike_encryption:'aes128',
						ike_dh_group:'modp1024',
						ike_authentication:'sha256',
						esp_encryption:'aes128',
						esp_dh_group:'modp1024',
						esp_authentication:'sha256',
						key_lefe:'360',
						key_lefe_type:'d',
						ike_lefe_time:'360',
						ike_lefe_time_type:'d',
						rekey_margin:'5',
						rekey_margin_type:'m',
						dpdaction:'restart',
						dpddelay:'30',
						dpddelay_type:'s',
						left_interface:'None',
						rekey: 'No',
						reauth:'No',
						forceencaps: 'No',
						mobike: 'No',
						used_source_ip: '' //注意页面无此参数
					};
				Object.assign(vm.addIPSecDialogForm,params);
				vm.$refs.addIPSecDialogForm.clearValidate();
			},
			addNrPlmnAddClick(){// 修改NR CELL弹窗中 plmnList 新增
				var vm = this, id = 'id', params = { custParamName: '', custParamValue: '', custParamPath: ''};
				if(vm.gnbAddOrEditForm.custParam.length == 0){
					params[id] = '1';
				}else{
					var idList=[];
					vm.gnbAddOrEditForm.custParam.map((item)=>{
						idList.push(item.id + '');
					})
					params.id = vm.createId(1,idList); 
					var delNum = 0;
					vm.gnbAddOrEditForm.custParam.map((item)=>{
						delNum+=1;
					})
				}
				vm.gnbAddOrEditForm.custParam.push(params);
				event.stopPropagation();
			},
			addNrPlmnListDel(row){//Customized Paeameters : delete
				var vm = this;
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					vm.gnbAddOrEditForm.custParam.map(function(item,index){
						if(item.id == row.id){
							vm.gnbAddOrEditForm.custParam = vm.gnbAddOrEditForm.custParam.filter((items)=>{
								return items.id != row.id
							})
						}
					})
				}).catch(()=>{})
			},
			gnbAdvanceSettingClick() {
				var vm = this;
				vm.gnbAdvanceTreeShow = true; //params advance tree
				vm.gnbAddVersionShow = false; //版本窗口
				vm.gnbLicenseImportShow = false; //license
				vm.commonImportFileShow = false; //params import
				vm.commonImportFileInfoShow = false; // params import info
				vm.commonAddOrUpdateParamShow = false; //参数配置 commonAddOrUpdateParam
				vm.addIPSecTunnelDialogShow = false; // ipsec
			},
			gnbAdvanceTreeConfirm() {
				var vm = this, checkedKeys = '';
				checkedKeys = vm.$refs.advanceTree.getCheckedKeys();
				vm.advanceTreeSelected = checkedKeys;
				vm.gnbAdvanceSettingCancel();
			},
			gnbAdvanceSettingCancel(){
				var vm = this;
				vm.gnbAdvanceTreeShow = false;
			},
			removeGroupItem(code) {
				var vm = this, idx = vm.advanceTreeSelected.indexOf(code);
				if(idx > -1) {
					vm.advanceTreeSelected.splice(idx,1);
					vm.$refs.advanceTree.setChecked(code,false);
					vm.gnbAdvanceTreeConfirm();
				}
			},
            gnbPolicyInit(type, row) {
                var vm = this, timeList =['GMT0','CET-1','EAT-3','WAT-1','CAT-2','EET-2','WET0WEST,M3.5.0,M10.5.0/3','CET-1CEST,M3.5.0,M10.5.0/3','SAST-2','WAT-1WAST,M9.1.0,M4.1.0','HST10HDT,M3.2.0,M11.1.0','AKST9AKDT,M3.2.0,M11.1.0','AST4','BRT3','ART3','PYT4PYST,M10.1.0/0,M3.4.0/0','EST5','COT5','VET4:30','GFT3','AMT4AMST,M10.3.0/0,M2.3.0/0','WGT3WGST,M3.5.0/-2,M10.5.0/-1','ECT5','GYT4','CST5CDT,M3.2.0/0,M11.1.0/1','BOT4','PET5','MST7MDT,M4.1.0,M10.5.0','PMST3PMDT,M3.2.0,M11.1.0','CST6CDT,M4.1.0,M10.5.0' ,'UYT3' ,'FNT2' ,'SRT3' ,'MST7' ,'AMT4','ACT5','PST8PDT,M4.1.0,M10.5.0','BRT3BRST,M10.3.0/0,M2.3.0/0','EGT1EGST,M3.5.0/0,M10.5.0/1','NST3:30NDT,M3.2.0,M11.1.0','CST6','EST5EDT,M3.2.0,M11.1.0','PST8PDT,M3.2.0,M11.1.0','CST6CDT,M3.2.0,M11.1.0','MST7MDT,M3.2.0,M11.1.0','DAVT-7','DDUT-10','MIST-11','MAWT-5','CLT3','ROTT3','SYOT-3','UTC0CEST-2,M3.5.0/1,M10.5.0/3','VOST-6','ALMT-6','EET-2EEST,M3.5.4/24,M10.5.5/1','ANAT-12','AQTT-5','TMT-5','AZT-4AZST,M3.5.0/4,M10.5.0/5','EET-2EEST,M3.5.0/0,M10.5.0/0','KGT-6','BNT-8','CHOT-8CHOST,M3.5.6,M9.5.6/0','EET-2EEST,M3.5.5/0,M10.5.5/0','BDT-6','TLT-9','TJT-5','EET-2EEST,M3.5.5/24,M10.3.6/144','HKT-8','HOVT-7HOVST,M3.5.6,M9.5.6/0','IRKT-8','WIT-9','IST-2IDT,M3.4.4/26,M10.5.0','AFT-4:30','PETT-12','PKT-5','NPT-5:45','IST-5:30','MYT-8','MAGT-10','WITA-8','PHT-8','GST-4','KRAT-7','NOVT-6','OMST-6','ORAT-5','WIB-7','KST-8:30','QYZT-6','MMT-6:30','AST-3','SAKT-10','KST-9','SGT-8','SRET-11','CST-8','UZT-5','GET-4','BTT-6','JST-9','ULAT-8ULAST,M3.5.6,M9.5.6/0','XJT-6','ICT-7','VLAT-10','YAKT-9','YEKT-5','AMT-4','AZOT1AZOST,M3.5.0/0,M10.5.0/1','AST4ADT,M3.2.0,M11.1.0','CVT1','GST2','FKST3','ACST-9:30ACDT,M10.1.0,M4.1.0/3','ACST-9:30','ACWST-8:45','AEST-10','LHST-10:30LHDT-11,M10.1.0,M4.1.0','AWST-8','AEST-10AEDT,M10.1.0,M4.1.0/3','EET-2EEST,M3.5.0,M10.5.0/3','GMT0IST,M3.5.0/1,M10.5.0','WET0WEST,M3.5.0/1,M10.5.0','GMT0BST,M3.5.0/1,M10.5.0','SAMT-4','MSK-3','EET-2EEST,M3.5.0/3,M10.5.0/4','IOT-6','CXT-7','CCT-6:30','TFT-5','SCT-4','MVT-5','MUT-4','RET-4','WSST-13WSDT,M9.5.0/3,M4.1.0/4','NZST-12NZDT,M9.5.0,M4.1.0/3','BST-11','CHAST-12:45CHADT,M9.5.0/2:45,M4.1.0/3:45','CHUT-10','EAST5','VUT-11','PHOT-13','TKT-13','FJT-12FJST,M11.1.0,M1.3.4/75','TVT-12','GALT6','GAMT9','SBT-11','HST10', 'LINT-14','KOST-11','MHT-12','MART9:30','NRT-12','NUT11','NFT-11:30','NCT-11','SST11','PWT-9','PST8','PONT-11','PGT-10','CKT10','ChST-10','TAHT10','GILT-12','TOT-13','WAKT-12' ,'WFT-12'];
				vm.gnbIsReadOnly = type == 'readonly';
                if(type == 'add'){
                	vm.gnbPolicyTitle = '<%=rb.getString("XinZengCeLue")%>';
                	vm.gnbOperType = 'add';
					vm.importParamsQuery.policyId = '${policyId}';
                }else if(type == 'modify'){
                	vm.gnbPolicyTitle = '<%=rb.getString("XiuGaiCeLue")%>';
                	vm.gnbOperType = 'modify';
					vm.importParamsQuery.policyId = row.policyId;
                }else{
                	vm.gnbPolicyTitle = '<%=rb.getString("ChaKanCeLue")%>';
                	vm.gnbOperType = 'view';
					vm.importParamsQuery.policyId = row.policyId;
                }
				vm.timeZoneList = timeList.map(item=>{//获取时区
					return{label:item,value:item}
				})
                axios.post("${ctx}/gnb/pnp/getProductTypeSelect.action").then(function(res){ // 获取Product Type
                	var data = res.data;
                	if(data.length > 0){
                		vm.gnbProductList = data;
    					if(type == 'add'){
    						vm.gnbAddOrEditForm.productType = vm.gnbProductList[0].value;
    					}
    					if(vm.gnbIsReadOnly || type == 'modify') {//详情或修改，获取详情信息
							vm.gnbPolicyId = row.policyId;
							if(row.switch == '0' || row.switch == null || row.switch == undefined){
								vm.gnbAddOrEditForm.policySwitch = '0';
							}else{
								vm.gnbAddOrEditForm.policySwitch = '1';
							}
							if(row.configEnable == '0' || row.configEnable == null || row.configEnable == undefined){
								vm.gnbAddOrEditForm.selfConfigEnable = '0';
							}else{
								vm.gnbAddOrEditForm.selfConfigEnable = '1';
							}
							Object.assign(vm.gnbAddOrEditForm, row);
							vm.gnbGetParamsInfo(row.policyId);	
							initForm(vm.$refs.gnbAddOrEditForm);					
    					}
                	}
				});
            },
            gnbGetParamsInfo(policyId) {//根据策略policyId获取升级策略信息
	            var vm = this, params = { policyId: policyId };
	            axios.post('${ctx}/gnb/pnp/getPnPUpgradePolicyInfo.action', stringify(params)).then(function(res){
					var data = res.data;
					if(data){
						vm.gnbAddOrEditForm.preserveSetting = data.preserveSetting; //是否保留配置
						vm.gnbAddOrEditForm.targetVersion = data.destVersion; //目标版本
						vm.gnbAddVersionBtnShow = false;
						vm.gnbResultOriginalVersionShow = true;
						if(data.originalVersion != null){
							if(data.originalVersion == 'all'){
								vm.specifyVersionType = '1';
							}else{
								var originlist = data.originalVersion,
								resultOriginlist = originlist.split(',');
								vm.gnbResultOriginalVersionList = resultOriginlist.map(function(item){ return {originalVersion: item};});
							}
						}
					}					
				});
				var commonParams = { policyId: policyId, serialNumber: 'default'}; // 批量配置传递 sn, 公共配置传递default
				//查询所有参数接口
				axios.post('${ctx}/gnb/pnp/getPnPPolicyConfigInfo.action', stringify(commonParams)).then(function(res){
					var data = res.data;
					if(data){
						if(data.plmn && data.plmn.length > 0){
							data.plmn.map((item,index)=>{
								if(item.plmnConfigList && item.plmnConfigList.length > 0){//判断 item.plmnConfigList 是否存在
									item.plmnConfigList.map((items,idx)=>{
										items['plmnNrIndex'] = item.index;//向item.plmnConfigList中增加 plmnNrIndex 属性，取值为 item.index	
										items.index = items.index + '';
										if(items.sliceList && items.sliceList.length > 0){//判断 item.plmnConfigList.sliceList 是否存在
											items.sliceList.map((itemss,idxs)=>{
												itemss.index = itemss.index + '';
											})
										}else{
											items['sliceList'] = [];
										}
									})
								}else{
									item['plmnConfigList'] = [];//如果 item.plmnConfigList 不存在，则向其添加
								} 
							})
						}
						Object.assign(vm.gnbAddOrEditForm, data);
						vm.advanceTreeSelected = data.advancedConfigOptions?data.advancedConfigOptions.split(','):[];
						vm.defaultRreeChecked = data.advancedConfigOptions?data.advancedConfigOptions.split(','):[];
					}					
				});
	        },
			gnbProductTypeChange(type) { //产品类型改变
				var vm = this;
				vm.gnbAddOrEditForm.originalVersion = '';
				vm.gnbResultOriginalVersionList = [];
				vm.gnbAddOrEditForm.targetVersion = '';
				vm.gnbTarVersionList = [];
				vm.gnbCommonOriginalVerList();
			},
        	gnbSelectMethodChange(val){//software upgrade,license,parameter config 三个模块切换时，使其打开的右侧内容关闭； 需校验内容是否发生变化
        		var vm = this;
        		vm.gnbCommonRightModelClose();
        	},
        	gnbClickTab(val){//parameter config 中的 parameter data pool, batch import, region deploying 三个模块切换时，使其打开的右侧内容关闭； 需校验内容是否发生变化
        		var vm = this;
				if(vm.gnbParameterConfigActive == 'gnbParamsImport'){
					vm.urlConfigPlan = '${ctx}/gnb/pnp/queryBasicConfigPageList.action';
				}
        		vm.gnbCommonRightModelClose();
			},
			gnbCommonRightModelClose(){//右侧弹窗内容关闭
				var vm = this;
				vm.gnbAddVersionShow = false; //版本窗口
				vm.gnbLicenseImportShow = false; //license
				vm.commonImportFileShow = false; //params import
				vm.commonImportFileInfoShow = false; // params import info
				vm.gnbAdvanceTreeShow = false; // params advance tree
				vm.commonAddOrUpdateParamShow = false; //参数配置 commonAddOrUpdateParam
				vm.addIPSecTunnelDialogShow = false; // ipsec
			},
			gnbAddVersionClick(){//新增版本按钮点击事件
				var vm = this;
				vm.gnbAddVersionShow = true; //版本窗口
				vm.gnbLicenseImportShow = false; //license
				vm.commonImportFileShow = false; //params import
				vm.commonImportFileInfoShow = false; // params import info
				vm.gnbAdvanceTreeShow = false; // params advance tree
				vm.commonAddOrUpdateParamShow = false; //参数配置 commonAddOrUpdateParam
				vm.addIPSecTunnelDialogShow = false; // ipsec
				vm.gnbCurSelectVersionData = [];
				vm.gnbSoftwareAddVersionForm.versionList = [];
				vm.$refs.gnbSoftwareOriginalVersionTable.clearSelection();
				vm.gnbCommonOriginalVerList();
			},
			gnbCommonOriginalVerList(){//根据已选产品类型加载对应的初始版本数据
				var vm = this;
                axios.post("${ctx}/gnb/pnp/queryOriginalUpgradeVersionPageList.action",stringify({// 根据产品类型回显对应初始版本
                	productType: vm.gnbAddOrEditForm.productType,
                    searchText: '',
                    page: 1,
                    rows: 50
                })).then(function(res){
					var data = res.data;
					if( data && data.rows.length > 0){
						vm.gnbSoftwareOriginalVersionList = data.rows;
					}
				});
			},
			gnbAddVersionBtn(){//手动添加 version
				var vm = this, versionText = vm.gnbSoftwareAddVersionForm.versionStr, str = '';
				if(versionText == ''){
					vm.gnbVersionErrorMessage = '<%=rb.getString("ShuRuBanBenHao")%>';
				}else{
					if(versionText){
						str = versionText;
						if(vm.gnbSoftwareAddVersionForm.versionList.indexOf(str) == -1){
							vm.gnbSoftwareAddVersionForm.versionList.push(str);
							vm.gnbSoftwareAddVersionForm.versionStr = '';
							vm.gnbVersionErrorMessage = '';
							vm.gnbVersionMessage = '';
							vm.$refs.gnbSoftwareAddVersionForm.validateField('itemTest');
						}else{
							vm.gnbVersionErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}else{
						vm.gnbVersionErrorMessage = '<%=rb.getString("ShuRuBanBenHao")%>';
					}
				}
			},
			gnbRemoveVersion(item){//删除version
				var vm = this, index = vm.gnbSoftwareAddVersionForm.versionList.indexOf(item);
				if(index !== -1){
					vm.gnbSoftwareAddVersionForm.versionList.splice(index,1)
				}
				vm.gnbVersionErrorMessage = '';
			},
			gnbVersionBatchSelect(selection){//列表选中
				var vm = this;
				vm.gnbCurSelectVersionData = selection;
			},
        	gnbAddVersionSubmit(){//add version submit
        		var vm = this, manualVersionList = vm.gnbSoftwareAddVersionForm.versionList, versionList = vm.gnbCurSelectVersionData.map(function(item){ return item.originalVersion});
        		if(manualVersionList.length == 0 && vm.gnbCurSelectVersionData.length == 0){
        			vm.gnbVersionMessage = '<%=rb.getString("ZhiShaoXuanZeYiZhongBanBenFangShi")%>';
        			return;
        		}else{
        			vm.gnbVersionMessage = '';
            		var mergeVersionList = [], mergelist = manualVersionList.concat(versionList);
            		for(var i =0, len = mergelist.length; i<len; i++){
            			if(mergeVersionList.indexOf(mergelist[i]) === -1){
            				mergeVersionList.push(mergelist[i])
            			}
            		}
            		var curList = mergeVersionList.map(function(item){
						return {originalVersion: item }
					});
            		if(vm.gnbResultOriginalVersionList.length > 0){
            			var length1 = vm.gnbResultOriginalVersionList.length;
            			var length2 = curList.length;
            			for (var i = 0; i<length1; i++){
            				for(var j= 0; j<length2; j++){
            					if(vm.gnbResultOriginalVersionList.length > 0){
            						if(vm.gnbResultOriginalVersionList[i]['originalVersion'] === curList[j]['originalVersion']){
            							vm.gnbResultOriginalVersionList.splice(i,1);
            							length1 --;
            						}
            					}
            				}
            			}
            			for(var n = 0; n< curList.length; n++){
            				vm.gnbResultOriginalVersionList.push(curList[n]);
            			}
            		}else{
            			vm.gnbResultOriginalVersionList = mergeVersionList.map(function(item){
							return {originalVersion: item }
						})
            		}
            		vm.gnbAddOrEditForm.originalVersion = vm.gnbResultOriginalVersionList.map(function(item){ return item.originalVersion;}).join(',');
            		vm.gnbAddVersionShow = false;
            		vm.gnbAddVersionBtnShow = false;
                    vm.gnbResultOriginalVersionShow = true;
        		}
        	},
        	gnbAddVersionCancel(){//add version cancel
        		var vm = this;
        		vm.gnbAddVersionShow = false;
        		vm.gnbCurSelectVersionData = [];
        		vm.$refs.gnbSoftwareAddVersionForm.resetFields();
        		vm.gnbSoftwareAddVersionForm.versionList = [];
        		vm.$refs.gnbSoftwareOriginalVersionTable.clearSelection();
        	},
        	gnbDeleteVersionItem(item){//从结果版本列表中删除
				var vm = this, index = vm.gnbResultOriginalVersionList.indexOf(item);
				if(index !== -1){
					vm.gnbResultOriginalVersionList.splice(index,1)
				}
			},
			gnbClearVersionBtnClick(){//clear 操作
				var vm = this;
				vm.gnbResultOriginalVersionList = [];
			},
       		gnbDeleteLicenseClick(row,evt){//license delete
				var vm = this, params = { fileNames : row.file_name }
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/cell/license/doClearLicenseFile.action",stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
	    						message: '<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					})
	    					vm.$refs.licenseTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch()
				vm.gnbLicenseImportShow = false;
			},
			gnbImportLicenseClick(){//导入
        		var vm = this;
				vm.gnbLicenseImportShow = true; //license
				vm.gnbAddVersionShow = false; //版本窗口
				vm.commonImportFileShow = false; //params import
				vm.commonImportFileInfoShow = false; // params import info
				vm.gnbAdvanceTreeShow = false; // params advance tree
				vm.commonAddOrUpdateParamShow = false; //参数配置 commonAddOrUpdateParam
				vm.addIPSecTunnelDialogShow = false; // ipsec
			},
			gnbMoreCheckFile(res,file){    //发送请求，校验device文件内容
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
					vm.gnbLicenseImportShow = false;
					vm.$refs.licenseTable.refresh();
					vm.gnbMoreCloseFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态
				var fileList = vm.$refs.moreUpload.uploadFile;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
			gnbMoreFileChange(file,fileList){//选择文件后，校验格式，并赋值页面显示
				var vm = this, fileIndex = file.name.lastIndexOf("."), fileType = file.name.substr(fileIndex + 1, file.name.length);
                if(['lic'].indexOf(fileType.toLowerCase()) === -1){
                    return false;
                }else{
                    vm.fileName += file.name+',';
                    vm.gnbMoreFileParams.FileName = file.name;
                }
			},
			gnbMoreFileSelect(){// 选择文件
				var vm =this;
	    	    vm.fileName = '';
				vm.$refs.moreUpload.clearFiles();
				vm.$refs['more_file_up'].click();
			},
			gnbMoreCloseFileSelect(){// 移除导入文件
				var vm = this;
				vm.fileName = '';
				vm.$refs.moreUpload.clearFiles();
			},
			gnbMoreFileRequest(file){//提交时匹配的文件及文件流
				var vm = this;
				vm.gnbFileData.push(file.file);
			},
	        gnbLicenseImportSubmit() {/*确定导入*/
				var vm = this, fileImportArr = [], productType = vm.gnbAddOrEditForm.productType;
				vm.$refs.gnbLicenseForm.validate((valid) => {
	                   if (valid){
	   					var fd = new FormData(),
	    					config = {
	    						headers: { 'Content-Type': 'multipart/form-data' }
	    					};
	                    vm.$refs.moreUpload.submit();
	    				vm.gnbUploadBtnDisabled = true;
	    				vm.gnbLicenseLoading = true;
	    				if(vm.gnbFileData.length > 0){
	    					vm.gnbFileData.forEach(file =>{
	    					    if(file.name.substr(file.name.lastIndexOf(".")) === '.lic'){
                                   var obj = {
                                        fileName: file.name,
                                        fielSize: file.size
                                   }
                                   fileImportArr.push(obj)
                                   fd.append('uploadFile',file);
                                }
		    				})
	    				}
	    				fd.append('fileArr',JSON.stringify(fileImportArr));//文件名
	    				fd.append('productType',productType);
	    				axios.post("${ctx}/cell/license/uploadLicenseFile.action",fd,config).then(function(res){
	    					if(res.data["success"]){
	    						vm.$message.success('<%=rb.getString("DaoRuChengGong")%>');
	    					    vm.$refs.licenseTable.refresh();
	    					}else{
	    						vm.$message.error(res.data["message"]);
	    					}
	    					vm.gnbLicenseImportCancel();
	    				})
	               	}
	           	})
			},
			gnbLicenseImportCancel(){// 关闭导入弹出框
				var vm = this;
				vm.gnbLicenseImportShow = false;
				vm.gnbUploadBtnDisabled = false;
				vm.gnbLicenseLoading = false;
				vm.fileList = [];
				vm.fileName = '';
				vm.$refs.gnbLicenseForm.resetFields();
				vm.gnbImportModeSelection = 'moreFile';
				vm.gnbFileData =[];
			},				 
			queryEnbLicense(text) {
				var vm = this;
				vm.gnbLicenseQuery.searchText = text;
				vm.gnbLicenseQuery.productType = this.gnbAddOrEditForm.productType;
			},
        	executeStatus(row,column,value,rowIndex){//字段格式化
				if ('0' == value) {
					return '<%=rb.getString("WeiZhiXing")%>';
				} else if ('1' == value){
					return '<%=rb.getString("ZhengZaiZhiXing")%>';
				} else if ('2' == value){
					return '<%=rb.getString("ZhiXingChengGong")%>';
				} else if ('3' == value){
					return '<%=rb.getString("ZhiXingShiBai")%>';
				}else{
					return '';
				}
			},		
            gnbAddOrUpdateSubmit() {
                var vm = this, params = {};
            	if(vm.gnbOperType == 'modify'){
            		params.policyId = vm.gnbPolicyId;
            	}else{
            		params.policyId = '${policyId}';
            	}				
				params.policySwitch = vm.gnbAddOrEditForm.policySwitch;
                params.policyName = encodeURIComponent(vm.gnbAddOrEditForm.policyName);
                params.productType = vm.gnbAddOrEditForm.productType;
                params.executeType = vm.gnbAddOrEditForm.executeType;
                params.upgradeEnable = vm.gnbAddOrEditForm.upgradeEnable;
                if(vm.specifyVersionType == '1'){
                	params.originalVersion = 'all';
                }else{
                	if(vm.gnbResultOriginalVersionList.length > 0){
                		params.originalVersion = vm.gnbResultOriginalVersionList.map(function(item){ return item.originalVersion;}).join(',');
                	}
                }
				params.targetVersion = vm.gnbAddOrEditForm.targetVersion;
                params.preserveSetting = vm.gnbAddOrEditForm.preserveSetting;
                params.licenseEnable = vm.gnbAddOrEditForm.licenseEnable;
				params.selfConfigEnable = vm.gnbAddOrEditForm.selfConfigEnable;
				params.paramConfigEnable = vm.gnbAddOrEditForm.paramConfigEnable;
                params.batchConfigEnable = vm.gnbAddOrEditForm.batchConfigEnable;
				params.gnbName = vm.gnbAddOrEditForm.gnbName;
                params.gnbLength = vm.gnbAddOrEditForm.gnbLength;
                params.gnbId = vm.gnbAddOrEditForm.gnbId;
				params.pci = vm.gnbAddOrEditForm.pci;
				params.freqBandIndicator = vm.gnbAddOrEditForm.freqBandIndicator;
				params.nrarfcndl = vm.gnbAddOrEditForm.nrarfcndl;
				params.dlbandwidth = vm.gnbAddOrEditForm.dlbandwidth;
				params.ssbFrequency = vm.gnbAddOrEditForm.ssbFrequency;
                params.nrarfcnul = vm.gnbAddOrEditForm.nrarfcnul;
				params.ulbandwidth = vm.gnbAddOrEditForm.ulbandwidth;
				params.duplex_mode = vm.gnbAddOrEditForm.duplex_mode;
				params.dlSubcarrierSpacing = vm.gnbAddOrEditForm.dlSubcarrierSpacing;
				params.ulSubcarrierSpacing = vm.gnbAddOrEditForm.ulSubcarrierSpacing;
				params.ulAntNum = vm.gnbAddOrEditForm.ulAntNum;
				params.dlAntNum = vm.gnbAddOrEditForm.dlAntNum;
				params.dlulTransmissionPeriodicity1 = vm.gnbAddOrEditForm.dlulTransmissionPeriodicity1;
				params.nrofDownlinkSlots1 = vm.gnbAddOrEditForm.nrofDownlinkSlots1;
				params.nrofUplinkSlots1 = vm.gnbAddOrEditForm.nrofUplinkSlots1;
				params.nrofDownlinkSymbols1 = vm.gnbAddOrEditForm.nrofDownlinkSymbols1;			
				params.nrofUplinkSymbols1 = vm.gnbAddOrEditForm.nrofUplinkSymbols1;
				params.dlulTransmissionPeriodicity2 = vm.gnbAddOrEditForm.dlulTransmissionPeriodicity2;
				params.nrofDownlinkSlots2 = vm.gnbAddOrEditForm.nrofDownlinkSlots2;
				params.nrofUplinkSlots2 = vm.gnbAddOrEditForm.nrofUplinkSlots2;
				params.nrofDownlinkSymbols2 = vm.gnbAddOrEditForm.nrofDownlinkSymbols2;
				params.nrofUplinkSymbols2 = vm.gnbAddOrEditForm.nrofUplinkSymbols2;
				params.prachRootSequenceIndex = vm.gnbAddOrEditForm.prachRootSequenceIndex;
                params.prachRootSequenceValue = vm.gnbAddOrEditForm.prachRootSequenceValue;
				params.plmn = JSON.stringify(vm.gnbAddOrEditForm.plmn);
				params.periodicInformEnable = vm.gnbAddOrEditForm.periodicInformEnable;
				params.periodicInformTime = vm.gnbAddOrEditForm.periodicInformTime;
				params.periodicInformInterval = vm.gnbAddOrEditForm.periodicInformInterval;
				params.ntpEnable = vm.gnbAddOrEditForm.ntpEnable;
				params.localTimeZone = vm.gnbAddOrEditForm.localTimeZone;
				params.ntpServer1 = vm.gnbAddOrEditForm.ntpServer1;
				params.ntpServer2 = vm.gnbAddOrEditForm.ntpServer2;
				params.ntpServer3 = vm.gnbAddOrEditForm.ntpServer3;
				params.ntpServer4 = vm.gnbAddOrEditForm.ntpServer4;
				params.ntpServer5 = vm.gnbAddOrEditForm.ntpServer5;
				params.advancedConfigOptions = vm.advanceTreeSelected.join(',');
				params.amf = JSON.stringify(vm.gnbAddOrEditForm.amf);
				params.wan = JSON.stringify(vm.gnbAddOrEditForm.wan);
				params.lan = JSON.stringify(vm.gnbAddOrEditForm.lan);				
				params.ipsecEnable = vm.gnbAddOrEditForm.ipsecEnable;
                params.ipsecImsi = vm.gnbAddOrEditForm.ipsecImsi;
                params.ipsecKey = vm.gnbAddOrEditForm.ipsecKey;
				params.ipsecOpc = vm.gnbAddOrEditForm.ipsecOpc;
                params.ipsecUsimEnable = vm.gnbAddOrEditForm.ipsecUsimEnable;
                params.ipsecUsimAuthEnable = vm.gnbAddOrEditForm.ipsecUsimAuthEnable;
				params.ipsecTunnel = JSON.stringify(vm.gnbAddOrEditForm.ipsecTunnel);
				params.staticRouting = JSON.stringify(vm.gnbAddOrEditForm.staticRouting);
				params.halobEnable = vm.gnbAddOrEditForm.halobEnable;
                params.halobMode = vm.gnbAddOrEditForm.halobMode;
				params.custParam = JSON.stringify(vm.gnbAddOrEditForm.custParam);
                vm.$refs.gnbAddOrEditForm.validate(function(valid){
					if(valid){
						vm.gnbSaveBtnDisabled = true;
                        axios.post('${ctx}/gnb/pnp/addPnPPolicy.action',stringify(params)).then(function(res){
                        	var data = res.data;
                        	if(data){
                        		 vm.gnbSaveBtnDisabled = false;
                        		if(data['success'] == true) {
                                    vm.$message({
    		    						message: '<%=rb.getString("ChengGong")%>',
    		    						type:'success',
    		    					});
                                    eventBus.$emit('gnb_reload-config-list');
                                    vm.closeConfig();
                                }else {
                                    vm.$message.error(data['message']);
                                }
                        	}
                        }).catch(function(){});
					}
				})
            },
			closeConfig() {
                eventBus.$emit('gnbClose-config');
            },
            gnbAddOrUpdateCancel(){
            	var vm = this;
				eventBus.$emit('gnbCancel-plugPlaySlide')
            },
			queryPlan(){
				Object.assign(this.importParamsQuery,this.importParamsQueryForm);
			},
			createId(idVal,list){
				var vm = this, val = idVal + '';
				if(list.includes(val) == true){
					idVal += 1 ;
					return vm.createId(idVal,list);
				}else{
					return  idVal + '';
				}
			},
			isValidIP(ip){//校验IP
				var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
				return reg.test(ip);     
			},
			isIPv6(str){ //Ipv6校验 
				var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
				return reg.test(str);
			},
			isMask(str){ //校验子网掩码
				var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
				return exp.test(str); 		
			},
			isNumeric(str) {// 验证输入的是否是数字
				if(str.length==0){ return false; }
				for(var i=0;i<str.length;i++){
					if(str.charAt(i)<"0" || str.charAt(i)>"9"){
						return false;
					}
				}
				return true;  
			}
        },
        mounted() {
        	var vm = this;
			eventBus.$off('gnbInit-plugPlayConfig').$on('gnbInit-plugPlayConfig', vm.gnbPolicyInit);
            eventBus.$off('gnbSave-plugPlayConfig').$on('gnbSave-plugPlayConfig', vm.gnbAddOrUpdateSubmit);
            eventBus.$off('gnbHandle-plugPlayCancel').$on('gnbHandle-plugPlayCancel',vm.gnbAddOrUpdateCancel);
        }
    });
</script>